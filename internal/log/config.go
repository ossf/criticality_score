package log

import (
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
