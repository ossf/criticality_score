package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/ossf/scorecard/v4/cron/data"
	"github.com/ossf/scorecard/v4/cron/worker"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector"
	"github.com/ossf/criticality_score/internal/scorer"
	"github.com/ossf/criticality_score/internal/signalio"
)

type collectWorker struct {
	logger          *zap.Logger
	c               *collector.Collector
	s               *scorer.Scorer
	scoreColumnName string
}

// Process implements the worker.Worker interface.
func (w *collectWorker) Process(ctx context.Context, req *data.ScorecardBatchRequest, bucketURL string) error {
	filename := worker.ResultFilename(req)

	// Prepare the logger with identifiers for the shard and job.
	logger := w.logger.With(
		zap.Int32("shard_id", req.GetShardNum()),
		zap.Time("job_time", req.GetJobTime().AsTime()),
		zap.String("filename", filename),
	)
	logger.Info("Processing shard")

	// Prepare the output writer
	extras := []string{}
	if w.s != nil {
		extras = append(extras, w.scoreColumnName)
	}
	var output bytes.Buffer
	out := signalio.CsvWriter(&output, w.c.EmptySets(), extras...)

	// Iterate through the repos in this shard.
	for _, repo := range req.GetRepos() {
		rawURL := repo.GetUrl()
		if rawURL == "" {
			logger.Warn("Skipping empty repo URL")
			continue
		}

		// Create a logger for this repo.
		repoLogger := logger.With(zap.String("repo", rawURL))
		repoLogger.Info("Processing repo")

		// Parse the URL to ensure it is a URL.
		u, err := url.Parse(rawURL)
		if err != nil {
			// TODO: record a metric
			repoLogger.With(zap.Error(err)).Warn("Failed to parse repo URL")
			continue
		}
		ss, err := w.c.Collect(ctx, u)
		if err != nil {
			if errors.Is(err, collector.ErrUncollectableRepo) {
				repoLogger.With(zap.Error(err)).Warn("Repo is uncollectable")
				continue
			}
			return fmt.Errorf("failed during signal collection: %w", err)
		}

		// If scoring is enabled, prepare the extra data to be output.
		extras := []signalio.Field{}
		if w.s != nil {
			f := signalio.Field{
				Key:   w.scoreColumnName,
				Value: fmt.Sprintf("%.5f", w.s.Score(ss)),
			}
			extras = append(extras, f)
		}

		// Write the signals to storage.
		if err := out.WriteSignals(ss, extras...); err != nil {
			return fmt.Errorf("failed writing signals: %w", err)
		}
	}

	// Write to the canonical bucket last. The presence of the file indicates
	// the job was completed. See scorecard's worker package for details.
	if err := data.WriteToBlobStore(ctx, bucketURL, filename, output.Bytes()); err != nil {
		return fmt.Errorf("error writing to blob store: %w", err)
	}

	logger.Info("Shard written successfully")

	return nil
}

func getScorer(logger *zap.Logger, scoringEnabled bool, scoringConfigFile string) (*scorer.Scorer, error) {
	if !scoringEnabled {
		logger.Info("Scoring: disabled")
		return nil, nil
	}
	if scoringConfigFile == "" {
		logger.Info("Scoring: using default config")
		return scorer.FromDefaultConfig(), nil
	}
	logger.With(zap.String("filename", scoringConfigFile)).Info("Scoring: using config file")

	f, err := os.Open(scoringConfigFile)
	if err != nil {
		return nil, fmt.Errorf("opening config: %w", err)
	}
	defer f.Close()

	s, err := scorer.FromConfig(scorer.NameFromFilepath(scoringConfigFile), f)
	if err != nil {
		return nil, fmt.Errorf("from config: %w", err)
	}
	return s, nil
}

func NewWorker(ctx context.Context, logger *zap.Logger, scoringEnabled bool, scoringConfigFile, scoringColumn string, collectOpts []collector.Option) (*collectWorker, error) {
	logger.Info("Initializing worker")
	logger.Debug("Creating collector")
	c, err := collector.New(ctx, logger, collectOpts...)
	if err != nil {
		return nil, fmt.Errorf("collector: %w", err)
	}

	logger.Debug("Creating scorer")
	s, err := getScorer(logger, scoringEnabled, scoringConfigFile)
	if err != nil {
		return nil, fmt.Errorf("scorer: %w", err)
	}

	// If we have the scorer, and the column isn't overridden, use the scorer's
	// name.
	if s != nil && scoringColumn == "" {
		scoringColumn = s.Name()
	}

	return &collectWorker{
		logger:          logger,
		c:               c,
		s:               s,
		scoreColumnName: scoringColumn,
	}, nil
}

// PostProcess implements the worker.Worker interface.
func (w *collectWorker) PostProcess() {}
