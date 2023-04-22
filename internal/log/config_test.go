package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"reflect"
	"testing"
)

func Test_dev(t *testing.T) {
	preSetConfig := zap.NewDevelopmentEncoderConfig()
	preSetConfig.CallerKey = ""
	test := struct {
		name  string
		want  zap.Config
		want1 []zap.Option
	}{
		name: "default",
		want: zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    preSetConfig,
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		},
		want1: []zap.Option{},
	}
	got, got1 := dev()

	if !ZapConfigEqual(got, test.want) {
		t.Errorf("dev() \ngot  = %v, \nwant = %v", got, test.want)
	}
	if !reflect.DeepEqual(got1, test.want1) {
		t.Errorf("dev() got1 = %v, want %v", got1, test.want1)
	}
}

func Test_gcp(t *testing.T) {
	test := struct { //nolint:govet
		name string
		want zap.Config
	}{
		name: "default",
		want: zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
			Development: false,
			Sampling:    nil,
			Encoding:    "json",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:       "timestamp",
				LevelKey:      "severity",
				NameKey:       "logger",
				CallerKey:     "caller",
				MessageKey:    "message",
				StacktraceKey: "stacktrace",
				LineEnding:    zapcore.DefaultLineEnding,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		},
	}

	got, _ := gcp()
	if !ZapConfigEqual(got, test.want) {
		t.Errorf("gcp() got  = %v, want = %v", got, test.want)
	}
}

func EncoderConfigEqual(got, want zapcore.EncoderConfig) bool {
	return got.MessageKey == want.MessageKey &&
		got.LevelKey == want.LevelKey &&
		got.TimeKey == want.TimeKey &&
		got.NameKey == want.NameKey &&
		got.CallerKey == want.CallerKey &&
		got.FunctionKey == want.FunctionKey &&
		got.StacktraceKey == want.StacktraceKey &&
		got.SkipLineEnding == want.SkipLineEnding &&
		got.LineEnding == want.LineEnding &&
		got.ConsoleSeparator == want.ConsoleSeparator &&
		reflect.DeepEqual(got.EncodeName, want.EncodeName) &&
		reflect.ValueOf(got.NewReflectedEncoder).Pointer() == reflect.ValueOf(want.NewReflectedEncoder).Pointer()
}

func ZapConfigEqual(x, y zap.Config) bool {
	if x.Encoding != y.Encoding {
		return false
	}
	if !EncoderConfigEqual(x.EncoderConfig, y.EncoderConfig) {
		return false
	}
	if x.Development != y.Development {
		return false
	}
	if x.DisableCaller != y.DisableCaller {
		return false
	}
	if x.DisableStacktrace != y.DisableStacktrace {
		return false
	}
	if !reflect.DeepEqual(x.InitialFields, y.InitialFields) {
		return false
	}
	if x.Sampling == nil || y.Sampling == nil {
		if x.Sampling != y.Sampling {
			return false
		}
	} else if !reflect.DeepEqual(*x.Sampling, *y.Sampling) {
		return false
	}
	if !reflect.DeepEqual(x.OutputPaths, y.OutputPaths) {
		return false
	}
	if !reflect.DeepEqual(x.ErrorOutputPaths, y.ErrorOutputPaths) {
		return false
	}
	if x.Level.Level() != y.Level.Level() {
		return false
	}
	return true
}
