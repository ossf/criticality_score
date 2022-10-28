package infile

import (
	"context"
	"io"
	"os"
)

// fileOpenFunc makes it possible to mock os.Open() for testing.
type fileOpenFunc func(string) (*os.File, error)

var (
	fileOpen fileOpenFunc = os.Open

	// The name that is used to represent stdin.
	StdinName = "-"
)

// Open opens and returns a file for input with the given filename.
//
// If filename is equal to o.StdoutName, os.Stdin will be used.
// If filename does not exist, an error will be returned.
// If filename does exist, the file will be opened and returned.
func Open(ctx context.Context, filename string) (io.ReadCloser, error) {
	if StdinName != "" && filename == StdinName {
		return os.Stdin, nil
	}
	return fileOpen(filename)
}
