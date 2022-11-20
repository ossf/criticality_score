// Copyright 2022 Criticality Score Authors
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

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"strings"

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/data"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"

	"github.com/ossf/criticality_score/internal/log"
)

const defaultLogLevel = zapcore.InfoLevel

func processShardSet(ctx context.Context, logger *zap.Logger, summary *data.ShardSummary, srcBucket, destBucket, destFilename string, threshold float64) (int, error) {
	logger = logger.With(zap.Time("creation_time", summary.CreationTime()))
	if summary.IsTransferred() || !summary.IsCompleted(threshold) {
		logger.With(
			zap.Bool("is_transferred", summary.IsTransferred()),
			zap.Bool("is_completed", summary.IsCompleted(threshold)),
		).Debug("Skipping")
		return 0, nil
	}

	logger.Info("Transferring...")

	// Scan the prefix for all the keys with a given prefix.
	// This will also include the
	keys, err := data.GetBlobKeysWithPrefix(ctx, srcBucket, data.GetBlobFilename("", summary.CreationTime()))
	if err != nil {
		return 0, fmt.Errorf("fetching blob keys by prefix: %w", err)
	}

	var out bytes.Buffer
	w := csv.NewWriter(&out)

	// Stores the first header row we encounter. It is used to generate the
	// header row for the aggregate CSV, and to validate all the CSV files
	// have the same columns.
	var firstHeader []string

	// Store the total number of records processed.
	var total int

	// Iterate through keys.
	for _, key := range keys {
		keyLogger := logger.With(zap.String("blob_key", key))

		_, filename, err := data.ParseBlobFilename(key)
		if err != nil {
			return 0, fmt.Errorf("parsing key %s into blob filename: %w", key, err)
		}

		// Filter out any keys that don't point to the shard files containing the data.
		if !strings.HasPrefix(filename, "shard-") {
			keyLogger.Debug("Skipping key")
			continue
		}

		keyLogger.Debug("Processing key")

		// Grab the shard CSV data.
		content, err := data.GetBlobContent(ctx, srcBucket, key)
		if err != nil {
			return 0, fmt.Errorf("fetching blob content for key %s: %w", key, err)
		}

		// Wrap the data in a buffer and CSV Reader to make it easy to consume.
		b := bytes.NewBuffer(content)
		r := csv.NewReader(b)

		header, err := r.Read()
		if err != nil {
			return 0, fmt.Errorf("reading header row for key %s: %w", key, err)
		}

		// Ensure the CSV headers are consistent.
		if firstHeader == nil {
			firstHeader = header
			if err := w.Write(header); err != nil {
				return 0, fmt.Errorf("writing header row from %s: %w", key, err)
			}
		}

		// Ensure the headers are consistent to avoid columns not matching.
		if !slices.Equal(firstHeader, header) {
			return 0, fmt.Errorf("key %s header does not match first header", key)
		}

		records, err := r.ReadAll()
		if err != nil {
			return 0, fmt.Errorf("reading all records for key %s: %w", key, err)
		}
		total += len(records)

		if err := w.WriteAll(records); err != nil {
			return 0, fmt.Errorf("writing all records for key %s: %w", key, err)
		}
	}

	// Store the aggregated CSV file.
	if err := data.WriteToBlobStore(ctx, destBucket, data.GetBlobFilename(destFilename, summary.CreationTime()), out.Bytes()); err != nil {
		return 0, fmt.Errorf("writing aggregate csv: %w", err)
	}

	// Mark the summary as completed so it doesn't get reprocessed again.
	if err := summary.MarkTransferred(ctx, srcBucket); err != nil {
		return 0, fmt.Errorf("marking shards as transferred: %w", err)
	}

	logger.With(zap.Int("total_records", total)).Info("Transfer complete")
	return total, nil
}

func main() {
	flag.Parse()

	if err := config.ReadConfig(); err != nil {
		panic(err)
	}

	// Extract the criticality score specific variables in the config.
	criticalityConfig, err := config.GetCriticalityValues()
	if err != nil {
		panic(err)
	}

	// Setup the logger.
	logger, err := log.NewLoggerFromConfigMap(log.DefaultEnv, defaultLogLevel, criticalityConfig)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	completionThreshold, err := config.GetCompletionThreshold()
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to get completion threshold")
	}

	// TODO: use a more sensible configuration entry.
	srcBucketURL, err := config.GetRawResultDataBucketURL()
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to get raw data bucket URL")
	}

	destBucketURL := criticalityConfig["csv-transfer-bucket-url"]
	if destBucketURL == "" {
		logger.Fatal("Failed to get CSV transfer bucket URL")
	}

	destFilename := criticalityConfig["csv-transfer-filename"]
	if destFilename == "" {
		logger.Fatal("Failed to get CSV transfer filename")
	}

	logger = logger.With(
		zap.String("src_bucket", srcBucketURL),
		zap.String("dest_bucket", destBucketURL),
		zap.String("dest_filename", destFilename),
	)

	ctx := context.Background()

	bucketSummary, err := data.GetBucketSummary(ctx, srcBucketURL)
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to get bucket summary")
	}
	shards := bucketSummary.Shards()

	numShardSets := len(shards)
	var processShardSets int
	var erroredShardSets int
	var totalRecords int

	logger.With(zap.Int("number_shard_sets", numShardSets)).Info("Found shards")

	for _, summary := range shards {
		if n, err := processShardSet(ctx, logger, summary, srcBucketURL, destBucketURL, destFilename, completionThreshold); err != nil {
			// Show an error, but continue processing.
			logger.With(
				zap.Error(err),
				zap.Time("creation_time", summary.CreationTime()),
			).Error("Failed to process shard set")
			erroredShardSets++
		} else if n > 0 {
			processShardSets++
			totalRecords += n
		}
	}

	// Output a bunch of useful metrics.
	logger.With(
		zap.Int("number_shard_sets", numShardSets),
		zap.Int("processed_shard_sets", processShardSets),
		zap.Int("errored_shard_sets", erroredShardSets),
		zap.Int("skipped_shard_sets", numShardSets-processShardSets-erroredShardSets),
		zap.Int("total_records_processed", totalRecords),
	).Info("Done")
}
