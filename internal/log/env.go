package log

import "errors"

type Env int

const (
	UnknownEnv = Env(0)
	DevEnv     = Env(iota)
	GCPEnv

	DefaultEnv = DevEnv
)

var ErrorUnkownEnv = errors.New("unknown logging environment")

// LookupEnv will return the instance of Env that corresponds to text.
//
// If text does not match a known environment UnknownEnv will be returned.
func LookupEnv(text string) Env {
	switch text {
	case "dev":
		return DevEnv
	case "gcp":
		return GCPEnv
	default:
		return UnknownEnv
	}
}

func (e Env) String() string {
	switch e {
	case DevEnv:
		return "dev"
	case GCPEnv:
		return "gcp"
	default:
		// panic?
		return "unknown"
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (e *Env) UnmarshalText(text []byte) error {
	*e = LookupEnv(string(text))
	if *e == UnknownEnv {
		return ErrorUnkownEnv
	}
	return nil
}

func (e Env) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}
