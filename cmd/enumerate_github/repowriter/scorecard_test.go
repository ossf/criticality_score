package repowriter_test

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ossf/criticality_score/cmd/enumerate_github/repowriter"
)

func TestScorecardRepoWriter(t *testing.T) {
	var buf bytes.Buffer
	w := repowriter.Scorecard(&buf)
	w.Write("https://github.com/example/example")
	w.Write("https://github.com/ossf/criticality_score")

	want := "repo,metadata\n" +
		"https://github.com/example/example,\n" +
		"https://github.com/ossf/criticality_score,\n"

	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Fatalf("Scorecard() mismatch (-want +got):\n%s", diff)
	}
}
