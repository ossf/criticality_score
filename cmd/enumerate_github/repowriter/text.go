package repowriter

import (
	"fmt"
	"io"
)

type textWriter struct {
	w io.Writer
}

// Text creates a new Writer instance that is used to write a simple text file
// of repositories, where each line has a single repository url.
func Text(w io.Writer) Writer {
	return &textWriter{w}
}

// Write implements the Writer interface.
func (w *textWriter) Write(repo string) error {
	_, err := fmt.Fprintln(w.w, repo)
	return err
}
