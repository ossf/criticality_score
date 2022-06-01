package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/ossf/criticality_score/cmd/enumerate_github/githubsearch"
	"github.com/ossf/criticality_score/internal/logflag"
	"github.com/ossf/criticality_score/internal/outfile"
	"github.com/ossf/criticality_score/internal/workerpool"
	"github.com/ossf/scorecard/v4/clients/githubrepo/roundtripper"
	sclog "github.com/ossf/scorecard/v4/log"
	"github.com/shurcooL/githubv4"
	log "github.com/sirupsen/logrus"
)

const (
	githubDateFormat = "2006-01-02"
	reposPerPage     = 100
	oneDay           = time.Hour * 24
	defaultLogLevel  = log.InfoLevel
)

var (
	// epochDate is the earliest date for which GitHub has data.
	epochDate = time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC)

	minStarsFlag        = flag.Int("min-stars", 10, "only enumerates repositories with this or more of stars.")
	starOverlapFlag     = flag.Int("star-overlap", 5, "the number of stars to overlap between queries.")
	requireMinStarsFlag = flag.Bool("require-min-stars", false, "abort if -min-stars can't be reached during enumeration.")
	queryFlag           = flag.String("query", "is:public", "sets the base query to use for enumeration.")
	workersFlag         = flag.Int("workers", 1, "the total number of concurrent workers to use.")
	startDateFlag       = dateFlag(epochDate)
	endDateFlag         = dateFlag(time.Now().UTC().Truncate(oneDay))
	logFlag             = logflag.Level(defaultLogLevel)
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
	return (time.Time)(*d)
}

// logLevelFlag implements the flag.Value interface to simplify the input and validation
// of the current log level.
type logLevelFlag log.Level

func (l *logLevelFlag) Set(value string) error {
	level, err := log.ParseLevel(string(value))
	if err != nil {
		return err
	}
	*l = logLevelFlag(level)
	return nil
}

func (l logLevelFlag) String() string {
	return log.Level(l).String()
}

func (l logLevelFlag) Level() log.Level {
	return log.Level(l)
}

func init() {
	flag.Var(&startDateFlag, "start", "the start `date` to enumerate back to. Must be at or after 2008-01-01.")
	flag.Var(&endDateFlag, "end", "the end `date` to enumerate from.")
	flag.Var(&logFlag, "log", "set the `level` of logging.")
	outfile.DefineFlags(flag.CommandLine, "force", "append", "FILE")
	flag.Usage = func() {
		cmdName := path.Base(os.Args[0])
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage:\n  %s [FLAGS]... FILE\n\n", cmdName)
		fmt.Fprintf(w, "Enumerates GitHub repositories between -start date and -end date, with -min-stars\n")
		fmt.Fprintf(w, "or higher. Writes each repository URL on a separate line to FILE.\n")
		fmt.Fprintf(w, "\nFlags:\n")
		flag.PrintDefaults()
	}
}

// searchWorker waits for a query on the queries channel, starts a search with that query using s
// and returns each repository on the results channel.
func searchWorker(s *githubsearch.Searcher, logger *log.Entry, queries, results chan string) {
	for q := range queries {
		total := 0
		err := s.ReposByStars(q, *minStarsFlag, *starOverlapFlag, func(repo string) {
			results <- repo
			total++
		})
		if err != nil {
			// TODO: this error handling is not at all graceful, and hard to recover from.
			logger.WithFields(log.Fields{
				"query": q,
				"error": err,
			}).Error("Enumeration failed for query")
			if errors.Is(err, githubsearch.ErrorUnableToListAllResult) {
				if *requireMinStarsFlag {
					os.Exit(1)
				}
			} else {
				os.Exit(1)
			}
		}
		logger.WithFields(log.Fields{
			"query":      q,
			"repo_count": total,
		}).Info("Enumeration for query done")
	}
}

func main() {
	flag.Parse()

	logger := log.New()
	logger.SetLevel(logFlag.Level())

	// roundtripper requires us to use the scorecard logger.
	scLogger := sclog.NewLogrusLogger(logger)

	// Ensure the -start date is not before the epoch.
	if startDateFlag.Time().Before(epochDate) {
		logger.Errorf("-start date must be no earlier than %s", epochDate.Format(githubDateFormat))
		os.Exit(2)
	}

	// Ensure -start is before -end
	if endDateFlag.Time().Before(startDateFlag.Time()) {
		logger.Error("-start date must be before -end date")
		os.Exit(2)
	}

	// Ensure a non-flag argument (the output file) is specified.
	if flag.NArg() != 1 {
		logger.Error("An output file must be specified.")
		os.Exit(2)
	}
	outFilename := flag.Arg(0)

	// Print a helpful message indicating the configuration we're using.
	logger.WithFields(log.Fields{
		"filename": outFilename,
	}).Info("Preparing output file")

	// Open the output file
	out, err := outfile.Open(outFilename)
	if err != nil {
		// File failed to open
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to open output file")
		os.Exit(2)
	}
	defer out.Close()

	logger.WithFields(log.Fields{
		"start":        startDateFlag.String(),
		"end":          endDateFlag.String(),
		"min_stars":    *minStarsFlag,
		"star_overlap": *starOverlapFlag,
		"workers":      *workersFlag,
	}).Info("Starting enumeration")

	// Track how long it takes to enumerate the repositories
	startTime := time.Now()
	ctx := context.Background()

	// Prepare a client for communicating with GitHub's GraphQL API
	rt := roundtripper.NewTransport(ctx, scLogger)
	httpClient := &http.Client{
		Transport: rt,
	}
	client := githubv4.NewClient(httpClient)

	baseQuery := *queryFlag
	queries := make(chan string)
	results := make(chan string, (*workersFlag)*reposPerPage)

	// Start the worker goroutines to execute the search queries
	wait := workerpool.WorkerPool(*workersFlag, func(i int) {
		workerLogger := logger.WithFields(log.Fields{"worker": i})
		s := githubsearch.NewSearcher(ctx, client, workerLogger, githubsearch.PerPage(reposPerPage))
		searchWorker(s, workerLogger, queries, results)
	})

	// Start a separate goroutine to collect results so worker output is always consumed.
	done := make(chan bool)
	totalRepos := 0
	go func() {
		for repo := range results {
			fmt.Fprintln(out, repo)
			totalRepos++
		}
		done <- true
	}()

	// Work happens here. Iterate through the dates from today, until the start date.
	for created := endDateFlag.Time(); !startDateFlag.Time().After(created); created = created.Add(-oneDay) {
		logger.WithFields(log.Fields{
			"created": created.Format(githubDateFormat),
		}).Info("Scheduling day for enumeration")
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

	logger.WithFields(log.Fields{
		"total_repos": totalRepos,
		"duration":    time.Now().Sub(startTime).Truncate(time.Minute).String(),
	}).Info("Finished enumeration")
}
