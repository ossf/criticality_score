// Copyright 2023 Criticality Score Authors
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

package log

import (
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	configLogEnvKey   = "log-env"
	configLogLevelKey = "log-level"
)

func dev() (zap.Config, []zap.Option) {
	c := zap.NewDevelopmentConfig()
	c.EncoderConfig.CallerKey = zapcore.OmitKey
	// TODO, use go-isatty to choose color VS no-color
	c.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	c.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	return c, []zap.Option{}
}

func gcp() (zap.Config, []zap.Option) {
	c := zapdriver.NewProductionConfig()
	// Make sure sampling is disabled.
	c.Sampling = nil
	// Build the logger and ensure we use the zapdriver Core so that labels
	// are handled correctly.
	return c, []zap.Option{zapdriver.WrapCore()}
}

// NewLogger returns a new instance of the zap.Logger based on the specified
// env and level.
//
// The level sets the minimum level log messages will be output, with
// env being used to configure the logger for a particular environment.
func NewLogger(e Env, l zapcore.Level) (*zap.Logger, error) {
	var c zap.Config
	var opts []zap.Option
	switch e {
	case GCPEnv:
		c, opts = gcp()
	default:
		c, opts = dev()
	}

	c.Level = zap.NewAtomicLevelAt(l)
	return c.Build(opts...)
}

// NewLoggerFromConfigMap returns a new instance of the zap.Logger based on
// the value of the keys "log-env" and "log-level" in the config map.
//
// If the "log-env" key is not present, defaultEnv will be used.
// If the "log-level" key is not present, defaultLevel will be used.
func NewLoggerFromConfigMap(defaultEnv Env, defaultLevel zapcore.Level, config map[string]string) (*zap.Logger, error) {
	// Extract the log environment from the config, if it exists.
	logEnv := defaultEnv
	if val := config[configLogEnvKey]; val != "" {
		if err := logEnv.UnmarshalText([]byte(val)); err != nil {
			return nil, err
		}
	}

	// Extract the log level from the config, if it exists.
	logLevel := defaultLevel
	if val := config[configLogLevelKey]; val != "" {
		if err := logLevel.Set(val); err != nil {
			return nil, err
		}
	}

	return NewLogger(logEnv, logLevel)
}
