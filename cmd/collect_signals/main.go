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
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/internal/collector"
	log "github.com/ossf/criticality_score/internal/log"
)

const (
	defaultLogLevel            = zapcore.InfoLevel
	githubAuthServerMaxAttemps = 10
)

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
	if commitID != "" {
		logger = logger.With(zap.String("commit_id", commitID))
	}

	// Extract the GCP project ID.
	gcpProjectID, err := config.GetProjectID()
	if err != nil {
		logger.With(zap.Error(err)).Fatal("Failed to get GCP Project ID")
	}

	// Extract the GCP dataset name.
	gcpDatasetName := criticalityConfig["dataset"]
	if gcpDatasetName == "" {
		gcpDatasetName = collector.DefaultGCPDatasetName
	}

	// Extract the GCP dataset TTL.
	gcpDatasetTTLHours := criticalityConfig["dataset-ttl-hours"]
	gcpDatasetTTL := time.Duration(0)
	if gcpDatasetTTLHours != "" {
		i, err := strconv.Atoi(gcpDatasetTTLHours)
		if err != nil {
			logger.With(zap.Error(err)).Fatal("Failed to get GCP Dataset TTL")
		}
		gcpDatasetTTL = time.Hour * time.Duration(i)
	}

	// Determine whether scoring is enabled or disabled.
	// It supports various "truthy" and "fasley" values. It will default to
	// enabled.
	scoringState := strings.ToLower(criticalityConfig["scoring"])
	scoringEnabled := true // this value is overridden below
	switch scoringState {
	case "", "yes", "enabled", "enable", "on", "true", "1":
		scoringEnabled = true
	case "no", "disabled", "disable", "off", "false", "0":
		scoringEnabled = false
	default:
		// Fatal exits.
		logger.Fatal("Unknown 'scoring' setting: " + scoringState)
	}

	scoringConfigFile := criticalityConfig["scoring-config"]
	scoringColumnName := criticalityConfig["scoring-column-name"]

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

	loop := worker.NewWorkLoop(w)
	if err := loop.Run(); err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Worker run loop failed")
	}

	logger.Info("Done.")
}
