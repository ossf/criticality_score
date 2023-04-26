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
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"github.com/ossf/scorecard/v4/clients/githubrepo/roundtripper"
	sclog "github.com/ossf/scorecard/v4/log"
	"github.com/shurcooL/githubv4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/cmd/enumerate_github/githubsearch"
	"github.com/ossf/criticality_score/cmd/enumerate_github/marker"
	"github.com/ossf/criticality_score/cmd/enumerate_github/repowriter"
	"github.com/ossf/criticality_score/internal/envflag"
	"github.com/ossf/criticality_score/internal/githubapi"
	log "github.com/ossf/criticality_score/internal/log"
	"github.com/ossf/criticality_score/internal/outfile"
	"github.com/ossf/criticality_score/internal/workerpool"
)

const (
	githubDateFormat = "2006-01-02"
	reposPerPage     = 100
	oneDay           = time.Hour * 24
	defaultLogLevel  = zapcore.InfoLevel
	runIDToken       = "[[runid]]"
	runIDDateFormat  = "20060102-1504"
)

var (
	// epochDate is the earliest date for which GitHub has data.
	epochDate = time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC)
	runID     = time.Now().UTC().Format(runIDDateFormat)

	markerFileFlag      = flag.String("marker", "", "writes the path to the file where results were written.")
	minStarsFlag        = flag.Int("min-stars", 10, "only enumerates repositories with this or more of stars.")
	starOverlapFlag     = flag.Int("star-overlap", 5, "the number of stars to overlap between queries.")
	requireMinStarsFlag = flag.Bool("require-min-stars", false, "abort if -min-stars can't be reached during enumeration.")
	queryFlag           = flag.String("query", "is:public", "sets the base query to use for enumeration.")
	workersFlag         = flag.Int("workers", 1, "the total number of concurrent workers to use.")
	startDateFlag       = dateFlag(epochDate)
	endDateFlag         = dateFlag(time.Now().UTC().Truncate(oneDay))
	logLevel            = defaultLogLevel
	logEnv              log.Env
	format              repowriter.WriterType
	markerType          marker.Type

	// Maps environment variables to the flags they correspond to.
	envFlagMap = envflag.Map{
		"CRITICALITY_SCORE_LOG_ENV":            "log-env",
		"CRITICALITY_SCORE_LOG_LEVEL":          "log",
		"CRITICALITY_SCORE_FORMAT":             "format",
		"CRITICALITY_SCORE_WORKERS":            "workers",
		"CRITICALITY_SCORE_START_DATE":         "start",
		"CRITICALITY_SCORE_END_DATE":           "end",
		"CRITICALITY_SCORE_OUTFILE":            "out",
		"CRITICALITY_SCORE_OUTFILE_FORCE":      "force",
		"CRITICALITY_SCORE_MARKER":             "marker",
		"CRITICALITY_SCORE_MARKER_TYPE":        "marker-type",
		"CRITICALITY_SCORE_QUERY":              "query",
		"CRITICALITY_SCORE_STARS_MIN":          "min-stars",
		"CRITICALITY_SCORE_STARS_OVERLAP":      "star-overlap",
		"CRITICALITY_SCORE_STARS_MIN_REQUIRED": "require-min-stars",
	}
)

// dateFlag implements the flag.Value interface to simplify the input and validation of
// dates from the command line.
type dateFlag time.Time

func (d *dateFlag) Set(value string) error {
	t, err := time.Parse(githubDateFormat, value)
	if err != nil {
		return err
	}
	*d = dateFlag(t)
	return nil
}

func (d *dateFlag) String() string {
	return (*time.Time)(d).Format(githubDateFormat)
}

func (d *dateFlag) Time() time.Time {
	return time.Time(*d)
}

func init() {
	flag.Var(&startDateFlag, "start", "the start `date` to enumerate back to. Must be at or after 2008-01-01.")
	flag.Var(&endDateFlag, "end", "the end `date` to enumerate from.")
	flag.Var(&logLevel, "log", "set the `level` of logging.")
	flag.TextVar(&format, "format", repowriter.WriterTypeText, "set output file `format`.")
	flag.TextVar(&logEnv, "log-env", log.DefaultEnv, "set logging `env`.")
	flag.TextVar(&markerType, "marker-type", marker.TypeFull, "format of the contents in the marker file. Can be 'full', 'dir' or 'file'.")
	outfile.DefineFlags(flag.CommandLine, "out", "force", "append", "FILE")
	flag.Usage = func() {
		cmdName := path.Base(os.Args[0])
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage:\n  %s [FLAGS]...\n\n", cmdName)
		fmt.Fprintf(w, "Enumerates GitHub repositories between -start date and -end date, with -min-stars\n")
		fmt.Fprintf(w, "or higher. Writes each repository URL in the specified format.\n")
		fmt.Fprintf(w, "\nFlags:\n")
		flag.PrintDefaults()
	}
}

// searchWorker waits for a query on the queries channel, starts a search with that query using s
// and returns each repository on the results channel.
func searchWorker(s *githubsearch.Searcher, logger *zap.Logger, queries, results chan string) {
	for q := range queries {
		total := 0
		err := s.ReposByStars(q, *minStarsFlag, *starOverlapFlag, func(repo string) {
			results <- repo
			total++
		})
		if err != nil {
			// TODO: this error handling is not at all graceful, and hard to recover from.
			logger.With(
				zap.String("query", q),
				zap.Error(err),
			).Error("Enumeration failed for query")
			if errors.Is(err, githubsearch.ErrorUnableToListAllResult) {
				if *requireMinStarsFlag {
					os.Exit(1)
				}
			} else {
				os.Exit(1)
			}
		}
		logger.With(
			zap.String("query", q),
			zap.Int("repo_count", total),
		).Info("Enumeration for query done")
	}
}

func main() {
	envflag.Parse(envFlagMap)

	logger, err := log.NewLogger(logEnv, logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Set a FilenameTransform to expand the run-id token in a filename.
	outfile.DefaultOpener.FilenameTransform = func(f string) string {
		if !strings.Contains(f, runIDToken) {
			return f
		}
		// Every future log message will have the run-id attached.
		logger = logger.With(zap.String("run-id", runID))
		logger.Info("Using Run ID")
		return strings.ReplaceAll(f, runIDToken, runID)
	}

	// roundtripper requires us to use the scorecard logger.
	innerLogger := zapr.NewLogger(logger)
	scLogger := &sclog.Logger{Logger: &innerLogger}

	// Warn if the -start date is before the epoch.
	if startDateFlag.Time().Before(epochDate) {
		logger.With(
			zap.String("start", startDateFlag.String()),
			zap.String("epoch", epochDate.Format(githubDateFormat)),
		).Warn("-start date is before epoch")
	}

	// Ensure -start is before -end
	if endDateFlag.Time().Before(startDateFlag.Time()) {
		logger.With(
			zap.String("start", startDateFlag.String()),
			zap.String("end", endDateFlag.String()),
		).Error("-start date must be before -end date")
		os.Exit(2)
	}

	// We need a context to support a bunch of operations.
	ctx := context.Background()

	// Prepare a client for communicating with GitHub's GraphQL API.
	// Do this before opening the output file to avoid creating an empty file
	// if we fail to authenticate, or connect to the authentication server.
	rt := githubapi.NewRetryRoundTripper(roundtripper.NewTransport(ctx, scLogger), logger)
	httpClient := &http.Client{
		Transport: rt,
	}
	client := githubv4.NewClient(httpClient)

	// Open the output file
	out, err := outfile.Open(ctx)
	if err != nil {
		// File failed to open
		logger.Error("Failed to open output file", zap.Error(err))
		os.Exit(2)
	}
	defer out.Close()
	w := format.New(out)

	logger.With(
		zap.String("start", startDateFlag.String()),
		zap.String("end", endDateFlag.String()),
		zap.Int("min_stars", *minStarsFlag),
		zap.Int("star_overlap", *starOverlapFlag),
		zap.Int("workers", *workersFlag),
	).Info("Starting enumeration")

	// Track how long it takes to enumerate the repositories
	startTime := time.Now()

	baseQuery := *queryFlag
	queries := make(chan string)
	results := make(chan string, (*workersFlag)*reposPerPage)

	// Start the worker goroutines to execute the search queries
	wait := workerpool.WorkerPool(*workersFlag, func(i int) {
		workerLogger := logger.With(zap.Int("worker", i))
		s := githubsearch.NewSearcher(ctx, client, workerLogger, githubsearch.PerPage(reposPerPage))
		searchWorker(s, workerLogger, queries, results)
	})

	// Start a separate goroutine to collect results so worker output is always consumed.
	done := make(chan bool)
	totalRepos := 0
	go func() {
		for repo := range results {
			w.Write(repo)
			totalRepos++
		}
		done <- true
	}()

	// Work happens here. Iterate through the dates from today, until the start date.
	for created := endDateFlag.Time(); !startDateFlag.Time().After(created); created = created.Add(-oneDay) {
		logger.With(
			zap.String("created", created.Format(githubDateFormat)),
		).Info("Scheduling day for enumeration")
		queries <- baseQuery + fmt.Sprintf(" created:%s", created.Format(githubDateFormat))
	}
	logger.Debug("Waiting for workers to finish")
	// Indicate to the workers that we're finished.
	close(queries)
	// Wait for the workers to be finished.
	wait()

	logger.Debug("Waiting for writer to finish")
	// Close the results channel now the workers are done.
	close(results)
	// Wait for the writer to be finished.
	<-done

	// Trigger Close() early to ensure the data exists before the marker file.
	if err := out.Close(); err != nil {
		logger.Fatal("Failed write data", zap.Error(err))
	}

	if *markerFileFlag != "" {
		logger = logger.With(zap.String("marker_filename", *markerFileFlag))
		logger.Debug("Writing the marker file")

		if err := marker.Write(ctx, markerType, *markerFileFlag, out.Name()); err != nil {
			logger.Error("Failed creating marker file", zap.Error(err))
			// Don't exit after a failure to create the marker file. Just fail
			// to write the marker file.
		}
	}

	logger.With(
		zap.Int("total_repos", totalRepos),
		zap.Duration("duration", time.Since(startTime).Truncate(time.Minute)),
		zap.String("filename", out.Name()),
	).Info("Finished enumeration")
}
