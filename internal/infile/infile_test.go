package infile

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestOpenStdin(t *testing.T) {
	origStdinName := StdinName
	defer func() { StdinName = origStdinName }()
	StdinName = "-stdin-"
	f, err := Open(context.Background(), "-stdin-")
	if err != nil {
		t.Fatalf("Open() == %v, want nil", err)
	}
	if f != os.Stdin {
		t.Fatal("Open() == not stdin, want stdin")
	}
}

func TestOpen(t *testing.T) {
	want := "path/to/file"
	got := ""
	fileOpen = func(filename string) (*os.File, error) {
		got = filename
		return &os.File{}, nil
	}

	f, err := Open(context.Background(), want)
	if err != nil {
		t.Fatalf("Open() == %v, want nil", err)
	}
	if f == nil {
		t.Fatal("Open() == nil, want a file")
	}
	if got != want {
		t.Fatalf("Open(%q) opened %q", want, got)
	}
}

func TestOpenError(t *testing.T) {
	want := errors.New("test error")
	fileOpen = func(filename string) (*os.File, error) {
		return nil, want
	}
	_, err := Open(context.Background(), "path/to/file")
	if err == nil {
		t.Fatalf("Open() is nil, want %v", want)
	}
	if !errors.Is(err, want) {
		t.Fatalf("Open() returned %v, want %v", err, want)
	}
}
