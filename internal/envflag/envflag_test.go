package envflag_test

import (
	"flag"
	"testing"

	"github.com/ossf/criticality_score/internal/envflag"
)

func TestEnvVarSet(t *testing.T) {
	expected := "value"
	t.Setenv("TEST_ENV_VAR", expected)

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	s := fs.String("test-flag", "default", "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, []string{}, m)
	if err != nil {
		t.Errorf("Assign() = %v, want nil", err)
	}
	if *s != expected {
		t.Errorf("flag = %v, want %v", *s, expected)
	}
}

func TestEnvVarMissing(t *testing.T) {
	expected := "default"

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	s := fs.String("test-flag", "default", "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, []string{}, m)
	if err != nil {
		t.Errorf("Assign() = %v, want nil", err)
	}
	if *s != expected {
		t.Errorf("flag = %v, want %v", *s, expected)
	}
}

func TestEnvVarEmpty(t *testing.T) {
	expected := "default"
	t.Setenv("TEST_ENV_VAR", "")

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	s := fs.String("test-flag", "default", "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, []string{}, m)
	if err != nil {
		t.Errorf("Assign() = %v, want nil", err)
	}
	if *s != expected {
		t.Errorf("flag = %v, want %v", *s, expected)
	}
}

func TestEnvAndFlagSet(t *testing.T) {
	expected := "another_value"
	args := []string{"-test-flag=" + expected}
	t.Setenv("TEST_ENV_VAR", "value")

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	s := fs.String("test-flag", "default", "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, args, m)
	if err != nil {
		t.Errorf("Assign() = %v, want nil", err)
	}
	if *s != expected {
		t.Errorf("flag = %v, want %v", *s, expected)
	}
}

func TestMissingFlag(t *testing.T) {
	t.Setenv("TEST_ENV_VAR", "value")

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, []string{}, m)
	if err == nil {
		t.Error("Assign() = nil, want an error")
	}
}

func TestInvalidValue(t *testing.T) {
	t.Setenv("TEST_ENV_VAR", "not_a_number")

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	fs.Int("test-flag", 42, "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	err := envflag.ParseFlagSet(fs, []string{}, m)
	if err == nil {
		t.Error("Assign() = nil, want an error")
	}
}

func TestParse(t *testing.T) {
	expected := "value"
	t.Setenv("TEST_ENV_VAR", expected)

	s := flag.String("test-flag", "default", "usage")
	m := envflag.Map{
		"TEST_ENV_VAR": "test-flag",
	}
	envflag.Parse(m)
	if *s != expected {
		t.Errorf("flag = %v, want %v", *s, expected)
	}
}
