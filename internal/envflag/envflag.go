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
	"errors"
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
			if errors.Is(err, flag.ErrHelp) {
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
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}
	flag.Parse()
}
