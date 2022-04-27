package logflag_test

import (
	"flag"
	"testing"

	"github.com/ossf/criticality_score/internal/logflag"
	"github.com/sirupsen/logrus"
)

func TestDefault(t *testing.T) {
	level := logflag.Level(logrus.ErrorLevel)
	if l := level.Level(); l != logrus.ErrorLevel {
		t.Fatalf("Level() == %v, want %v", l, logrus.ErrorLevel)
	}
}

func TestSet(t *testing.T) {
	level := logflag.Level(logrus.InfoLevel)
	err := level.Set("error")
	if err != nil {
		t.Fatalf("Set() == %v, want nil", err)
	}
	if l := level.Level(); l != logrus.ErrorLevel {
		t.Fatalf("Level() == %v, want %v", l, logrus.ErrorLevel)
	}
}

func TestSetError(t *testing.T) {
	level := logflag.Level(logrus.InfoLevel)
	err := level.Set("hello,world")
	if err == nil {
		t.Fatalf("Set() == nil, want an error")
	}
}

func TestString(t *testing.T) {
	level := logflag.Level(logrus.DebugLevel)
	if s := level.String(); s != logrus.DebugLevel.String() {
		t.Fatalf("String() == %v, want %v", s, logrus.DebugLevel.String())
	}
}

func TestFlagUnset(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	level := logflag.Level(logrus.InfoLevel)
	fs.Var(&level, "level", "usage")
	err := fs.Parse([]string{"arg"})
	if err != nil {
		t.Fatalf("Parse() == %v, want nil", err)
	}
	if l := level.Level(); l != logrus.InfoLevel {
		t.Fatalf("Level() == %v, want %v", l, logrus.InfoLevel)
	}
}

func TestFlagSet(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	level := logflag.Level(logrus.InfoLevel)
	fs.Var(&level, "level", "usage")
	err := fs.Parse([]string{"-level=fatal", "arg"})
	if err != nil {
		t.Fatalf("Parse() == %v, want nil", err)
	}
	if l := level.Level(); l != logrus.FatalLevel {
		t.Fatalf("Level() == %v, want %v", l, logrus.FatalLevel)
	}
}

func TestFlagSetError(t *testing.T) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	level := logflag.Level(logrus.InfoLevel)
	fs.Var(&level, "level", "usage")
	err := fs.Parse([]string{"-level=foobar", "arg"})
	if err == nil {
		t.Fatalf("Parse() == nil, want an error")
	}
}
