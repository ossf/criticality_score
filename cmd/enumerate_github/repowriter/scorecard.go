package repowriter

import (
	"encoding/csv"
	"io"
)

var header = []string{"repo", "metadata"}

type scorecardWriter struct {
	w *csv.Writer
}

// Scorecard creates a new Writer instance that is used to write a csv file
// of repositories that is compatible with the github.com/ossf/scorecard
// project.
//
// The csv file has a header row with columns "repo" and "metadata". Each
// row consists of the repository url and blank metadata.
func Scorecard(w io.Writer) Writer {
	csvWriter := csv.NewWriter(w)
	csvWriter.Write(header)
	return &scorecardWriter{w: csvWriter}
}

// Write implements the Writer interface.
func (w *scorecardWriter) Write(repo string) error {
	if err := w.w.Write([]string{repo, ""}); err != nil {
		return err
	}
	w.w.Flush()
	return w.w.Error()
}
