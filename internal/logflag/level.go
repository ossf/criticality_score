// Package logflag is a simple helper library that generalizes the logic for
// parsing command line flags for configuring the logging behavior.
package logflag

import log "github.com/sirupsen/logrus"

// Level implements the flag.Value interface to simplify the input and validation
// of the current logrus log level.
//
//     var logLevel = logflag.Level(logrus.InfoLevel)
//     flag.Var(&logLevel, "log", "set the `level` of logging.")
type Level log.Level

// Set implements the flag.Value interface.
func (l *Level) Set(value string) error {
	level, err := log.ParseLevel(string(value))
	if err != nil {
		return err
	}
	*l = Level(level)
	return nil
}

// String implements the flag.Value interface.
func (l Level) String() string {
	return log.Level(l).String()
}

// Level returns either the default log level, or the value set on the command line.
func (l Level) Level() log.Level {
	return log.Level(l)
}
