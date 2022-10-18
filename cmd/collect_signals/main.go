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

	"github.com/ossf/criticality_score/cmd/collect_signals/result"
	"github.com/ossf/criticality_score/internal/collector"
	"github.com/ossf/criticality_score/internal/infile"
	log "github.com/ossf/criticality_score/internal/log"
	"github.com/ossf/criticality_score/internal/outfile"
	"github.com/ossf/criticality_score/internal/workerpool"
)

const defaultLogLevel = zapcore.InfoLevel

var (
	gcpProjectFlag     = flag.String("gcp-project-id", "", "the Google Cloud Project ID to use. Auto-detects by default.")
	depsdevDisableFlag = flag.Bool("depsdev-disable", false, "disables the collection of signals from deps.dev.")
	depsdevDatasetFlag = flag.String("depsdev-dataset", collector.DefaultGCPDatasetName, "the BigQuery dataset name to use.")
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

func handleRepo(ctx context.Context, logger *zap.Logger, c *collector.Collector, u *url.URL, out result.Writer) {
	ss, err := c.Collect(ctx, u)
	if err != nil {
		if errors.Is(err, collector.ErrUncollectableRepo) {
			logger.With(
				zap.Error(err),
			).Warn("Repo cannot be collected")
			return
		}
		logger.With(
			zap.Error(err),
		).Error("Failed to collect signals for repo")
		os.Exit(1) // TODO: pass up the error
	}

	rec := out.Record()
	for _, s := range ss {
		if err := rec.WriteSignalSet(s); err != nil {
			logger.With(
				zap.Error(err),
			).Error("Failed to write signal set")
			os.Exit(1) // TODO: pass up the error
		}
	}
	if err := rec.Done(); err != nil {
		logger.With(
			zap.Error(err),
		).Error("Failed to complete record")
		os.Exit(1) // TODO: pass up the error
	}
}

func main() {
	flag.Parse()

	logger, err := log.NewLogger(logEnv, logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Complete the validation of args
	if flag.NArg() != 2 {
		logger.Error("Must have one input file and one output file specified.")
		os.Exit(2)
	}

	ctx := context.Background()

	// Bump the # idle conns per host
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = *workersFlag * 5

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

	// Prepare the output writer
	out := result.NewCsvWriter(w, c.EmptySets())

	// Start the workers that process a channel of repo urls.
	repos := make(chan *url.URL)
	wait := workerpool.WorkerPool(*workersFlag, func(worker int) {
		innerLogger := logger.With(zap.Int("worker", worker))
		for u := range repos {
			handleRepo(ctx, innerLogger.With(zap.String("url", u.String())), c, u, out)
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
