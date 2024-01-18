// Copyright 2023 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package localworker

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/data"
	"github.com/ossf/scorecard/v4/cron/worker"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/iterator"
)

// maxAttempts is the maximum number of times a batch will be attempted before
// giving up and moving on to the next batch.
//
// TODO: make this configurable.
const maxAttempts = 7

const (
	configLocalURLDataFile = "local-url-data-file"
	configLocalStateFile   = "local-state-file"
)

// WorkLoop is a version of worker.WorkLoop that reads data from a local file, rather
// than a from a pubsub feed populated by the Scorecard controller.
type WorkLoop struct {
	logger *zap.Logger
	w      worker.Worker
	input  iterator.IterCloser[string]

	// stateFilename points to the file used to store progress to allow for
	// recovery if the worker is terminated for any reason. The name should not
	// change for a given input.
	stateFilename string

	// bucketURL is a gocloud.dev URL pointing to the data bucket where data
	// will be written.
	bucketURL string

	// rawBucketURL is a gocloud.dev URL pointing to the raw data bucket where
	// raw data will be written.
	rawBucketURL string

	// shardSize defines the size of each shard passed to the worker for
	// processing.
	shardSize int
}

// FromConfig returns a new local WorkLoop instance using the config values
// read in.
//
// Apart from the usual config values for the data bucket, raw data bucket, and
// shard size, the criticality score config values "local-url-data-file" and
// "local-state-file" are also used.
func FromConfig(logger *zap.Logger, w worker.Worker) (*WorkLoop, error) {
	if !flag.Parsed() {
		flag.Parse()
	}

	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	bucketURL, err := config.GetResultDataBucketURL()
	if err != nil {
		return nil, fmt.Errorf("config data bucket url: %w", err)
	}

	rawBucketURL, err := config.GetRawResultDataBucketURL()
	if err != nil {
		return nil, fmt.Errorf("config raw data bucket url: %w", err)
	}
	shardSize, err := config.GetShardSize()
	if err != nil {
		return nil, fmt.Errorf("config shard size: %w", err)
	}

	criticalityConfig, err := config.GetCriticalityValues()
	if err != nil {
		return nil, fmt.Errorf("criticality score config: %w", err)
	}

	inFile := criticalityConfig[configLocalURLDataFile]
	if inFile == "" {
		return nil, fmt.Errorf("%q config not set", configLocalURLDataFile)
	}
	// This file is eventually closed when the wrapping iterator has its Close
	// method called.
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("os.Open %s: %w", inFile, err)
	}

	stateFile := criticalityConfig[configLocalStateFile]
	if stateFile == "" {
		return nil, fmt.Errorf("%q config not set", configLocalStateFile)
	}

	return &WorkLoop{
		logger: logger,
		w:      w,

		input:         iterator.Lines(f),
		stateFilename: stateFile,

		bucketURL:    bucketURL,
		rawBucketURL: rawBucketURL,
		shardSize:    shardSize,
	}, nil
}

func (l *WorkLoop) restore(shards iterator.IterCloser[[]string], currenShard int32) error {
	if currenShard <= 0 {
		// No recovery needed if we're at the start.
		return nil
	}

	// If we are restoring, fast forward to the position we were at prior to starting.

	l.logger.With(zap.Int32("shard", currenShard)).Info("Restoring previous position")
	lastFinishedShard := currenShard - 1

	shard := int32(0)
	for shards.Next() {
		shards.Item()
		if shard >= lastFinishedShard {
			break
		}
		shard++
	}
	if err := shards.Err(); err != nil {
		return fmt.Errorf("restore state iterator: %w", err)
	}
	if shard < lastFinishedShard {
		return fmt.Errorf("restore state shard mismatch: got = %d; want = %d", shard, lastFinishedShard)
	}
	return nil
}

func (l *WorkLoop) process(ctx context.Context, req *data.ScorecardBatchRequest, bucketURL string) error {
	exists, err := resultExists(ctx, req, bucketURL)
	if err != nil {
		return fmt.Errorf("result exists check: %w", err)
	}

	// Sanity check - make sure we are not re-processing an already processed request.
	if exists {
		l.logger.With(zap.Int32("shard", req.GetShardNum())).Info("Shard already exists. Skipping.")
		return nil
	}

	if err := l.w.Process(ctx, req, bucketURL); err != nil {
		return fmt.Errorf("worker process: %w", err)
	}
	return nil
}

// Run iterates through the repositories in batches, calling the supplied
// worker.Process method for each batch.
func (l *WorkLoop) Run() error {
	ctx := context.Background()

	s, err := loadState(l.stateFilename)
	if err != nil {
		return fmt.Errorf("loadState: %w", err)
	}

	// Create the shard iterator
	shards := iterator.Batch(l.input, l.shardSize)

	// Restore the shard iterator position based on the current shard number.
	if err := l.restore(shards, s.Shard); err != nil {
		return fmt.Errorf("recovering state: %w", err)
	}

	l.logger.Info("Starting worker loop")

	for shards.Next() {
		req := makeRequest(shards.Item(), s.Shard, s.JobTime)

		logger := l.logger.With(zap.Int32("shard", s.Shard))
		logger.Info("Received batch subscription")

		for s.Attempt < maxAttempts {
			// Increment and save the attempt before executing so we know we
			// tried it before we failed.
			s.Attempt++
			if err := s.Save(); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}

			if err := l.process(ctx, req, l.bucketURL); err != nil {
				// This is the equivalent of a Nack in the PubSub handling.
				// However, since this worker is entirely self containerd we
				// merely try again immediately.
				// In the future we could improve this behavior and move retry
				// attempts to the end of the queue to add a delay. Additionally,
				// if different requests fail consecutively then returning an
				// error here may be better.
				logger.With(zap.Error(err), zap.Int("attempt", s.Attempt)).Info("Error processing request")
				continue
			}

			l.w.PostProcess()
			break
		}

		// Reset the number of attempts.
		s.Attempt = 0
		s.Shard++
	}

	if err := shards.Err(); err != nil {
		return fmt.Errorf("iterator: %w", err)
	}
	// Closing the iterator releases all the resources held by it.
	if err := shards.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	// Write out the metadata
	if err := writeMetadata(ctx, l.bucketURL, s.Shard, s.JobTime); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	if err := writeMetadata(ctx, l.rawBucketURL, s.Shard, s.JobTime); err != nil {
		return fmt.Errorf("writing metdata to raw: %w", err)
	}

	if err := s.Clear(); err != nil {
		return fmt.Errorf("clearing state: %w", err)
	}
	return nil
}
