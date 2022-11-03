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
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/internal/collector"
	log "github.com/ossf/criticality_score/internal/log"
	"github.com/ossf/criticality_score/internal/outfile"
	"github.com/ossf/criticality_score/internal/scorer"
	"github.com/ossf/criticality_score/internal/signalio"
	"github.com/ossf/criticality_score/internal/workerpool"
)

const defaultLogLevel = zapcore.InfoLevel

var (
	gcpProjectFlag        = flag.String("gcp-project-id", "", "the Google Cloud Project ID to use. Auto-detects by default.")
	depsdevDisableFlag    = flag.Bool("depsdev-disable", false, "disables the collection of signals from deps.dev.")
	depsdevDatasetFlag    = flag.String("depsdev-dataset", collector.DefaultGCPDatasetName, "the BigQuery dataset name to use.")
	scoringDisableFlag    = flag.Bool("scoring-disable", false, "disables the generation of scores.")
	scoringConfigFlag     = flag.String("scoring-config", "", "path to a YAML file for configuring the scoring algorithm.")
	scoringColumnNameFlag = flag.String("scoring-column", "", "manually specify the name for the column used to hold the score.")
	workersFlag           = flag.Int("workers", 1, "the total number of concurrent workers to use.")
	logLevel              = defaultLogLevel
	logEnv                log.Env
)

// initFlags prepares any runtime flags, usage information and parses the flags.
func initFlags() {
	flag.Var(&logLevel, "log", "set the `level` of logging.")
	flag.TextVar(&logEnv, "log-env", log.DefaultEnv, "set logging `env`.")
	outfile.DefineFlags(flag.CommandLine, "out", "force", "append", "OUTFILE")
	flag.Usage = func() {
		cmdName := path.Base(os.Args[0])
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage:\n  %s [FLAGS]... {FILE|REPO...}\n\n", cmdName)
		fmt.Fprintf(w, "Collects signals for a list of project repository urls.\n\n")
		fmt.Fprintf(w, "FILE must be either a file or - to read from stdin. If FILE does not\n")
		fmt.Fprintf(w, "exist it will be treated as a REPO.\n")
		fmt.Fprintf(w, "Each REPO must be a project repository url.\n")
		fmt.Fprintf(w, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
}

// getScorer prepares a Scorer based on the flags passed to the command.
//
// nil will be returned if scoring is disabled.
func getScorer(logger *zap.Logger) *scorer.Scorer {
	if *scoringDisableFlag {
		logger.Info("Scoring disabled")
		return nil
	}
	if *scoringConfigFlag == "" {
		logger.Info("Preparing default scorer")
		return scorer.FromDefaultConfig()
	}
	// Prepare the scorer from the config file
	logger = logger.With(
		zap.String("filename", *scoringConfigFlag),
	)
	logger.Info("Preparing scorer from config")
	cf, err := os.Open(*scoringConfigFlag)
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to open scoring config file")
		os.Exit(2)
	}
	defer cf.Close()

	s, err := scorer.FromConfig(scorer.NameFromFilepath(*scoringConfigFlag), cf)
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to initialize scorer")
		os.Exit(2)
	}
	return s
}

func generateScoreColumnName(s *scorer.Scorer) string {
	if s == nil {
		return ""
	}
	if *scoringColumnNameFlag != "" {
		return *scoringColumnNameFlag
	}
	return s.Name()
}

func main() {
	initFlags()

	logger, err := log.NewLogger(logEnv, logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Prepare the scorer, if it is enabled.
	s := getScorer(logger)
	scoreColumnName := generateScoreColumnName(s)

	// Complete the validation of args
	if flag.NArg() == 0 {
		logger.Error("An input file or at least one repo must be specified.")
		os.Exit(2)
	}

	ctx := context.Background()

	// Bump the # idle conns per host
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = *workersFlag * 5

	opts := []collector.Option{
		collector.EnableAllSources(),
		collector.GCPProject(*gcpProjectFlag),
		collector.GCPDatasetName(*depsdevDatasetFlag),
	}
	if *depsdevDisableFlag {
		opts = append(opts, collector.DisableSource(collector.SourceTypeDepsDev))
	}

	c, err := collector.New(ctx, logger, opts...)
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to create collector")
		os.Exit(2)
	}

	// Prepare the input for reading
	inputIter, err := initInput(flag.Args())
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to prepare input")
		os.Exit(2)
	}
	defer inputIter.Close()

	// Open the out-file for writing
	w, err := outfile.Open(context.Background())
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to open file for output")
		os.Exit(2)
	}
	defer w.Close()

	// Prepare the output writer
	extras := []string{}
	if s != nil {
		extras = append(extras, scoreColumnName)
	}
	out := signalio.CsvWriter(w, c.EmptySets(), extras...)

	// Start the workers that process a channel of repo urls.
	repos := make(chan *url.URL)
	wait := workerpool.WorkerPool(*workersFlag, func(worker int) {
		innerLogger := logger.With(zap.Int("worker", worker))
		for u := range repos {
			l := innerLogger.With(zap.String("url", u.String()))
			ss, err := c.Collect(ctx, u)
			if err != nil {
				if errors.Is(err, collector.ErrUncollectableRepo) {
					l.With(
						zap.Error(err),
					).Warn("Repo cannot be collected")
					return
				}
				l.With(
					zap.Error(err),
				).Error("Failed to collect signals for repo")
				os.Exit(1) // TODO: pass up the error
			}

			// If scoring is enabled, prepare the extra data to be output.
			extras := []signalio.Field{}
			if s != nil {
				f := signalio.Field{
					Key:   scoreColumnName,
					Value: fmt.Sprintf("%.5f", s.Score(ss)),
				}
				extras = append(extras, f)
			}

			// Write the signals to storage.
			if err := out.WriteSignals(ss, extras...); err != nil {
				l.With(
					zap.Error(err),
				).Error("Failed to write signal set")
				os.Exit(1) // TODO: pass up the error
			}
		}
	})

	// Read in each repo from the input
	for inputIter.Next() {
		line := inputIter.Item()

		u, err := url.Parse(strings.TrimSpace(line))
		if err != nil {
			logger.With(
				zap.String("url", line),
				zap.Error(err),
			).Error("Failed to parse project url")
			os.Exit(1) // TODO: add a flag to continue or abort on failure
		}
		logger.With(
			zap.String("url", u.String()),
		).Debug("Parsed project url")

		// Send the url to the workers
		repos <- u
	}
	if err := inputIter.Err(); err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed while reading input")
		os.Exit(2)
	}
	// Close the repos channel to indicate that there is no more input.
	close(repos)

	// Wait until all the workers have finished.
	wait()

	// TODO: track metrics as we are running to measure coverage of data
}
