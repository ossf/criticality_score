package repowriter_test

import (
	"errors"
	"testing"

	"github.com/ossf/criticality_score/v2/cmd/enumerate_github/repowriter"
)

func TestTypeString(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name       string
		writerType repowriter.WriterType
		want       string
	}{
		{name: "text", writerType: repowriter.WriterTypeText, want: "text"},
		{name: "scorecard", writerType: repowriter.WriterTypeScorecard, want: "scorecard"},
		{name: "unknown", writerType: repowriter.WriterType(10), want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.writerType.String()
			if got != test.want {
				t.Fatalf("String() == %s, want %s", got, test.want)
			}
		})
	}
}

func TestTypeMarshalText(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name       string
		writerType repowriter.WriterType
		want       string
		err        error
	}{
		{name: "text", writerType: repowriter.WriterTypeText, want: "text"},
		{name: "scorecard", writerType: repowriter.WriterTypeScorecard, want: "scorecard"},
		{name: "unknown", writerType: repowriter.WriterType(10), want: "", err: repowriter.ErrorUnknownRepoWriterType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.writerType.MarshalText()
			if err != nil && !errors.Is(err, test.err) {
				t.Fatalf("MarhsalText() == %v, want %v", err, test.err)
			}
			if err == nil {
				if test.err != nil {
					t.Fatalf("MarshalText() return nil error, want %v", test.err)
				}
				if string(got) != test.want {
					t.Fatalf("MarhsalText() == %s, want %s", got, test.want)
				}
			}
		})
	}
}

func TestTypeUnmarshalText(t *testing.T) {
	//nolint:govet
	tests := []struct {
		input string
		want  repowriter.WriterType
		err   error
	}{
		{input: "text", want: repowriter.WriterTypeText},
		{input: "scorecard", want: repowriter.WriterTypeScorecard},
		{input: "", want: 0, err: repowriter.ErrorUnknownRepoWriterType},
		{input: "unknown", want: 0, err: repowriter.ErrorUnknownRepoWriterType},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var got repowriter.WriterType
			err := got.UnmarshalText([]byte(test.input))
			if err != nil && !errors.Is(err, test.err) {
				t.Fatalf("UnmarshalText() == %v, want %v", err, test.err)
			}
			if err == nil {
				if test.err != nil {
					t.Fatalf("MarshalText() return nil error, want %v", test.err)
				}
				if got != test.want {
					t.Fatalf("UnmarshalText() parsed %d, want %d", int(got), int(test.want))
				}
			}
		})
	}
}
