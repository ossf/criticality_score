// Package envflag is a simple library for associating environment variables with flags.
//
// If using the default flag.CommandLine FlagSet, just call envflag.Parse() instead of flag.Parse().
//
// Assign environment variables to flags using the Map type:
//
//	var m := envflag.Map{
//	    "MY_ENV_VAR": "my-flag"
//	}
//
// If the flag and the environment variable is set the flag takes precidence.
package envflag

import (
	"flag"
	"os"
)

type Map map[string]string

func (m Map) Assign(fs *flag.FlagSet) error {
	for env, f := range m {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			err := fs.Set(f, v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ParseFlagSet(fs *flag.FlagSet, args []string, m Map) error {
	err := m.Assign(fs)
	if err != nil {
		switch fs.ErrorHandling() {
		case flag.ContinueOnError:
			return err
		case flag.ExitOnError:
			if err == flag.ErrHelp {
				os.Exit(0)
			}
			os.Exit(2)
		case flag.PanicOnError:
			panic(err)
		}
	}
	return fs.Parse(args)
}

func Parse(m Map) {
	if err := m.Assign(flag.CommandLine); err != nil {
		// flag.CommandLine is set for ExitOnError
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	flag.Parse()
}
