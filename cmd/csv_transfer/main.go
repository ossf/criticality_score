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

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/data"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"

	"github.com/ossf/criticality_score/internal/log"
)

const defaultLogLevel = zapcore.InfoLevel

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
	bucketURL, err := config.GetRawResultDataBucketURL()
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to get raw data bucket URL")
	}

	outBucketURL := criticalityConfig["csv-transfer-bucket-url"]
	if outBucketURL == "" {
		logger.Fatal("Failed to get CSV transfer bucket URL")
	}

	csvFilename := criticalityConfig["csv-transfer-filename"]
	if csvFilename == "" {
		logger.Fatal("Failed to get CSV transfer filename")
	}

	logger = logger.With(
		zap.String("src_bucket", bucketURL),
		zap.String("dest_bucket", outBucketURL),
		zap.String("filename", csvFilename),
	)

	ctx := context.Background()

	bucketSummary, err := data.GetBucketSummary(ctx, bucketURL)
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to get bucket summary")
	}
	shards := bucketSummary.Shards()

	// Create CSV output. This buffer will grown to many tens of MB in size.
	// We use the same buffer to avoid realloc.
	var out bytes.Buffer

	logger.With(zap.Int("total_shards", len(shards))).Info("Starting transfers")

OUTER:
	for _, summary := range shards {
		summaryLogger := logger.With(zap.Time("creation_time", summary.CreationTime()))
		if summary.IsTransferred() || !summary.IsCompleted(completionThreshold) {
			summaryLogger.With(
				zap.Bool("is_transferred", summary.IsTransferred()),
				zap.Bool("is_completed", summary.IsCompleted(completionThreshold)),
			).Debug("Skipping")
			continue
		}

		summaryLogger.Info("Transferring...")

		// Transfer shard summary...
		keys, err := data.GetBlobKeysWithPrefix(ctx, bucketURL, data.GetBlobFilename("", summary.CreationTime()))
		if err != nil {
			// Fatal exits.
			summaryLogger.With(zap.Error(err)).Fatal("Failed to get keys with prefix")
		}

		// Reset the buffer and prepare for writing.
		out.Reset()
		w := csv.NewWriter(&out)

		// Iterate through keys.
		var firstHeader []string
		for _, key := range keys {
			keyLogger := summaryLogger.With(zap.String("blob_key", key))
			keyLogger.Debug("Processing key")

			// Grab the shard CSV data.
			content, err := data.GetBlobContent(ctx, bucketURL, key)
			if err != nil {
				keyLogger.With(zap.Error(err)).Error("Failed to get blob content")
				continue OUTER
			}

			// Wrap the data in a buffer and CSV Reader to make it easy to consume.
			b := bytes.NewBuffer(content)
			r := csv.NewReader(b)

			header, err := r.Read()
			if err != nil {
				keyLogger.With(zap.Error(err)).Error("Failed to read header row")
				continue OUTER
			}

			// Ensure the CSV headers are consistent.
			if firstHeader == nil {
				firstHeader = header
				if err := w.Write(header); err != nil {
					keyLogger.With(zap.Error(err)).Error("Failed to write header row")
					continue OUTER
				}
			}

			// Ensure the headers are consistent to avoid columns not matching.
			if !slices.Equal(firstHeader, header) {
				keyLogger.Error("Header mismatch")
				continue OUTER
			}

			records, err := r.ReadAll()
			if err != nil {
				keyLogger.Error("Failed to read all csv records")
				continue OUTER
			}

			if err := w.WriteAll(records); err != nil {
				keyLogger.Error("Failed to write all csv records")
				continue OUTER
			}
		}

		// Store the aggregated CSV file.
		if err := data.WriteToBlobStore(ctx, outBucketURL, data.GetBlobFilename(csvFilename, summary.CreationTime()), out.Bytes()); err != nil {
			summaryLogger.With(zap.Error(err)).Error("Failed to write result csv to bucket")
			continue
		}

		// Mark the summary as completed so it doesn't get reprocessed again.
		if err := summary.MarkTransferred(ctx, bucketURL); err != nil {
			summaryLogger.With(zap.Error(err)).Error("Failed to mark shards as transferred")
			continue
		}

		summaryLogger.Info("Transfer complete")
	}

	logger.Info("Completed all transfers")
}
