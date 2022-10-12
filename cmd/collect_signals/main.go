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
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-logr/zapr"
	"github.com/ossf/scorecard/v4/clients/githubrepo/roundtripper"
	sclog "github.com/ossf/scorecard/v4/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/cmd/collect_signals/collector"
	"github.com/ossf/criticality_score/cmd/collect_signals/depsdev"
	"github.com/ossf/criticality_score/cmd/collect_signals/github"
	"github.com/ossf/criticality_score/cmd/collect_signals/githubmentions"
	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/cmd/collect_signals/result"
	"github.com/ossf/criticality_score/internal/githubapi"
	"github.com/ossf/criticality_score/internal/infile"
	log "github.com/ossf/criticality_score/internal/log"
	"github.com/ossf/criticality_score/internal/outfile"
	"github.com/ossf/criticality_score/internal/workerpool"
)

const defaultLogLevel = zapcore.InfoLevel

var (
	gcpProjectFlag     = flag.String("gcp-project-id", "", "the Google Cloud Project ID to use. Auto-detects by default.")
	depsdevDisableFlag = flag.Bool("depsdev-disable", false, "disables the collection of signals from deps.dev.")
	depsdevDatasetFlag = flag.String("depsdev-dataset", depsdev.DefaultDatasetName, "the BigQuery dataset name to use.")
	workersFlag        = flag.Int("workers", 1, "the total number of concurrent workers to use.")
	logLevel           = defaultLogLevel
	logEnv             log.Env
)

func init() {
	flag.Var(&logLevel, "log", "set the `level` of logging.")
	flag.TextVar(&logEnv, "log-env", log.DefaultEnv, "set logging `env`.")
	outfile.DefineFlags(flag.CommandLine, "force", "append", "OUT_FILE")
	flag.Usage = func() {
		cmdName := path.Base(os.Args[0])
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage:\n  %s [FLAGS]... IN_FILE OUT_FILE\n\n", cmdName)
		fmt.Fprintf(w, "Collects signals for each project repository listed.\n")
		fmt.Fprintf(w, "IN_FILE must be either a file or - to read from stdin.\n")
		fmt.Fprintf(w, "OUT_FILE must be either be a file or - to write to stdout.\n")
		fmt.Fprintf(w, "\nFlags:\n")
		flag.PrintDefaults()
	}
}

func handleRepo(ctx context.Context, logger *zap.Logger, u *url.URL, out result.Writer) {
	r, err := projectrepo.Resolve(ctx, u)
	if err != nil {
		logger.With(zap.Error(err)).Warn("Failed to create project")
		// TODO: we should have an error that indicates that the URL/Project
		// should be skipped/ignored.
		return // TODO: add a flag to continue or abort on failure
	}
	logger = logger.With(zap.String("canonical_url", r.URL().String()))

	// TODO: p.URL() should be checked to see if it has already been processed.

	// Collect the signals for the given project
	logger.Info("Collecting")
	ss, err := collector.Collect(ctx, r)
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to collect signals for project")
		os.Exit(1) // TODO: add a flag to continue or abort on failure
	}

	rec := out.Record()
	for _, s := range ss {
		if err := rec.WriteSignalSet(s); err != nil {
			logger.With(
				zap.Error(err),
			).Error("Failed to write signal set")
			os.Exit(1) // TODO: add a flag to continue or abort on failure
		}
	}
	if err := rec.Done(); err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to complete record")
		os.Exit(1) // TODO: add a flag to continue or abort on failure
	}
}

func initCollectors(ctx context.Context, logger *zap.Logger, ghClient *githubapi.Client) error {
	collector.Register(&github.RepoCollector{})
	collector.Register(&github.IssuesCollector{})
	collector.Register(githubmentions.NewCollector(ghClient))

	if *depsdevDisableFlag {
		// deps.dev collection has been disabled, so skip it.
		logger.Warn("deps.dev signal collection is disabled.")
	} else {
		ddcollector, err := depsdev.NewCollector(ctx, logger, *gcpProjectFlag, *depsdevDatasetFlag)
		if err != nil {
			return fmt.Errorf("init deps.dev collector: %w", err)
		}
		logger.Info("deps.dev signal collector enabled")
		collector.Register(ddcollector)
	}

	return nil
}

func main() {
	flag.Parse()

	logger, err := log.NewLogger(logEnv, logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// roundtripper requires us to use the scorecard logger.
	innerLogger := zapr.NewLogger(logger)
	scLogger := &sclog.Logger{Logger: &innerLogger}

	if flag.NArg() != 2 {
		logger.Error("Must have one input file and one output file specified.")
		os.Exit(2)
	}
	inFilename := flag.Args()[0]
	outFilename := flag.Args()[1]

	// Open the in-file for reading
	r, err := infile.Open(context.Background(), inFilename)
	if err != nil {
		logger.With(
			zap.String("filename", inFilename),
			zap.Error(err),
		).Error("Failed to open an input file")
		os.Exit(2)
	}
	defer r.Close()

	// Open the out-file for writing
	w, err := outfile.Open(context.Background(), outFilename)
	if err != nil {
		logger.With(
			zap.String("filename", outFilename),
			zap.Error(err),
		).Error("Failed to open file for output")
		os.Exit(2)
	}
	defer w.Close()

	ctx := context.Background()

	// Bump the # idle conns per host
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = *workersFlag * 5

	// Prepare a client for communicating with GitHub's GraphQLv4 API and Restv3 API
	rt := githubapi.NewRoundTripper(roundtripper.NewTransport(ctx, scLogger), logger)
	httpClient := &http.Client{
		Transport: rt,
	}
	ghClient := githubapi.NewClient(httpClient)

	// Register all the Repo factories.
	projectrepo.Register(github.NewRepoFactory(ghClient, logger))

	// Register all the collectors that are supported.
	err = initCollectors(ctx, logger, ghClient)
	if err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to initalize collectors")
		os.Exit(2)
	}

	// Prepare the output writer
	out := result.NewCsvWriter(w, collector.EmptySets())

	// Start the workers that process a channel of repo urls.
	repos := make(chan *url.URL)
	wait := workerpool.WorkerPool(*workersFlag, func(worker int) {
		innerLogger := logger.With(zap.Int("worker", worker))
		for u := range repos {
			handleRepo(ctx, innerLogger.With(zap.String("url", u.String())), u, out)
		}
	})

	// Read in each line from the input files
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

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
	if err := scanner.Err(); err != nil {
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
