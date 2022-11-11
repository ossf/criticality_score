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
	"strings"

	"github.com/ossf/scorecard/v4/cron/config"
	"github.com/ossf/scorecard/v4/cron/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ossf/criticality_score/internal/collector"
	log "github.com/ossf/criticality_score/internal/log"
	"github.com/ossf/criticality_score/internal/signalio"
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

	// Extract the log environment from the config, if it exists.
	logEnv := log.DefaultEnv
	if val := criticalityConfig["log-env"]; val != "" {
		if err := logEnv.UnmarshalText([]byte(val)); err != nil {
			panic(err)
		}
	}

	// Extract the log level from the config, if it exists.
	logLevel := defaultLogLevel
	if val := criticalityConfig["log-level"]; val != "" {
		if err := logLevel.Set(val); err != nil {
			panic(err)
		}
	}

	// Setup the logger.
	logger, err := log.NewLogger(logEnv, logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Parse the output format.
	formatType := signalio.WriterTypeCSV
	if val := criticalityConfig["output-format"]; val != "" {
		if err := formatType.UnmarshalText([]byte(val)); err != nil {
			panic(err)
		}
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

	// TODO: capture metrics similar to scorecard/cron/worker

	// Bump the # idle conns per host
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 5

	// Preapre the options for the collector.
	opts := []collector.Option{
		collector.EnableAllSources(),
		collector.GCPProject(gcpProjectID),
		collector.GCPDatasetName(gcpDatasetName),
	}

	w, err := NewWorker(context.Background(), logger, formatType, scoringEnabled, scoringConfigFile, scoringColumnName, opts)
	if err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Failed to create worker")
	}

	loop := worker.NewWorkLoop(w)
	if err := loop.Run(); err != nil {
		// Fatal exits.
		logger.With(zap.Error(err)).Fatal("Worker run loop failed")
	}

	logger.Info("Done.")
}
