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
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/cmd/collect_signals/localworker"
	"github.com/ossf/criticality_score/cmd/collect_signals/vcs"
	"github.com/ossf/criticality_score/internal/collector"
	log "github.com/ossf/criticality_score/internal/log"
)

const (
	defaultLogLevel            = zapcore.InfoLevel
	githubAuthServerMaxAttemps = 10
)

const (
	configDataset           = "dataset"
	configDatasetTTLHours   = "dataset-ttl-hours"
	configLocal             = "local"
	configScoring           = "scoring"
	configScoringConfigFile = "scoring-config"
	configScoringColumnName = "scoring-column-name"
)

type runner interface {
	Run() error
}

// getWorkLoop chooses between either scorecard's worker.WorkLoop implementation
// or the internal localworker.WorkLoop implementation. The localworker.WorkLoop
// is chosen if the "local" criticality config variable is enabled.
func getWorkLoop(logger *zap.Logger, criticalityConfig map[string]string, w worker.Worker) (runner, error) {
	local, err := parseBool(criticalityConfig[configLocal], false)
	if err != nil {
		return nil, fmt.Errorf("parsing %q config as bool: %w", configLocal, err)
	}
	if !local {
		wl := worker.NewWorkLoop(w)
		return &wl, nil
	}
	wl, err := localworker.FromConfig(logger, w)
	if err != nil {
		return nil, fmt.Errorf("localworker.NewWorkLoop: %w", err)
	}
	return wl, nil
}

// parseBool converts boolStr into a bool value. It supports converting various
// "truthy" and "falsey" values.
//
// If the empty string is encountered emptyVal is returned.
// If a bool value cannot be determined, an error will be returned.
func parseBool(boolStr string, emptyVal bool) (bool, error) {
	switch strings.ToLower(boolStr) {
	case "":
		return emptyVal, nil
	case "yes", "enabled", "enable", "on", "true", "1":
		return true, nil
	case "no", "disabled", "disable", "off", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool string '%s'", boolStr)
	}
}

func waitForRPCServer(logger *zap.Logger, rpcServer string, maxAttempts int) {
	if rpcServer == "" {
		return
	}

	logger = logger.With(zap.String("rpc_server", rpcServer))

	retryWait := time.Second
	attempt := 0
	for {
		attempt++
		c, err := rpc.DialHTTP("tcp", rpcServer)
		switch {
		case err == nil:
			c.Close()
			logger.Info("GitHub auth server found.")
			return
		case attempt < maxAttempts:
			logger.With(
				zap.Error(err),
				zap.Duration("wait", retryWait),
				zap.Int("attempt", attempt),
			).Warn("Waiting for GitHub auth server.")
		default:
			// Fatal exits.
			logger.With(
				zap.Error(err),
			).Fatal("Unable to find GitHub auth server.")
		}
		time.Sleep(retryWait)
		retryWait = retryWait + (retryWait / 2)
	}
}

func main() {
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

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

	// Embed the commitID with all log messages.
	if commitID := vcs.CommitID(); commitID != vcs.MissingCommitID {
		logger = logger.With(zap.String("commit_id", commitID))
	}

	// Extract the GCP project ID.
	gcpProjectID, err := config.GetProjectID()
	if err != nil {
		logger.With(zap.Error(err)).Fatal("Failed to get GCP Project ID")
	}

	// Extract the GCP dataset name.
	gcpDatasetName := criticalityConfig[configDataset]
	if gcpDatasetName == "" {
		gcpDatasetName = collector.DefaultGCPDatasetName
	}

	// Extract the GCP dataset TTL.
	gcpDatasetTTLHours := criticalityConfig[configDatasetTTLHours]
	gcpDatasetTTL := time.Duration(0)
	if gcpDatasetTTLHours != "" {
		i, err := strconv.Atoi(gcpDatasetTTLHours)
		if err != nil {
			logger.With(zap.Error(err)).Fatal("Failed to get GCP Dataset TTL")
		}
		gcpDatasetTTL = time.Hour * time.Duration(i)
	}

	// Determine whether scoring is enabled or disabled.
	scoringEnabled, err := parseBool(criticalityConfig[configScoring], true)
	if err != nil {
		logger.With(zap.Error(err)).Fatal(fmt.Sprintf("Failed parsing %q setting", configScoring))
	}

	scoringConfigFile := criticalityConfig[configScoringConfigFile]
	scoringColumnName := criticalityConfig[configScoringColumnName]

	// Extract the CSV bucket URL. Currently uses the raw result bucket url.
	// TODO: use a more sensible configuration entry.
	csvBucketURL, err := config.GetRawResultDataBucketURL()
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to read CSV bucket URL. Ignoring.")
		csvBucketURL = ""
	}

	// The GitHub authentication server may be unavailable if it is starting
	// at the same time. Wait until it can be reached.
	waitForRPCServer(logger, os.Getenv("GITHUB_AUTH_SERVER"), githubAuthServerMaxAttemps)

	// Bump the # idle conns per host
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 5

	// Preapre the options for the collector.
	opts := []collector.Option{
		collector.EnableAllSources(),
		collector.GCPProject(gcpProjectID),
		collector.GCPDatasetName(gcpDatasetName),
		collector.GCPDatasetTTL(gcpDatasetTTL),
	}

	w, err := NewWorker(context.Background(), logger, scoringEnabled, scoringConfigFile, scoringColumnName, csvBucketURL, opts)
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to create worker")
	}
	defer w.Close()

	loop, err := getWorkLoop(logger, criticalityConfig, w)
	if err != nil {
		logger.With(zap.Error(err)).Fatal("Failed to create work loop")
	}
	if err := loop.Run(); err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Worker run loop failed")
	}

	logger.Info("Done.")
}
