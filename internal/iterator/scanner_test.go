package iterator_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/exp/slices"

	"github.com/ossf/criticality_score/internal/iterator"
)

func TestScannerIter_Empty(t *testing.T) {
	var b bytes.Buffer
	i := iterator.Lines(io.NopCloser(&b))

	if got := i.Next(); got {
		t.Errorf("Next() = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
}

func TestScannerIter_SingleLine(t *testing.T) {
	want := "test line"
	b := bytes.NewBuffer([]byte(want))
	i := iterator.Lines(io.NopCloser(b))

	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
	if got := i.Item(); got != want {
		t.Errorf("Item() = %v; want %v", got, want)
	}
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
	}
}

func TestScannerIter_MultiLine(t *testing.T) {
	want := []string{"line one", "line two", "line three"}
	b := bytes.NewBuffer([]byte(strings.Join(want, "\n")))
	i := iterator.Lines(io.NopCloser(b))

	var got []string
	for i.Next() {
		item := i.Item()
		got = append(got, item)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Iterator returned %v, want %v", got, want)
	}
}

func TestScannerIter_Error(t *testing.T) {
	want := errors.New("error")
	r := iotest.ErrReader(want)
	i := iterator.Lines(io.NopCloser(r))

	if got := i.Next(); got {
		t.Errorf("Next() = %v; want false", got)
	}
	if err := i.Err(); err == nil || !errors.Is(err, want) {
		t.Errorf("Err() = %v; want %v", err, want)
	}
}

type closerFn func() error

func (c closerFn) Close() error {
	return c()
}

func TestScannerIter_Close(t *testing.T) {
	got := 0
	i := iterator.Lines(&struct {
		closerFn
		io.Reader
	}{
		closerFn: closerFn(func() error {
			got++
			return nil
		}),
		Reader: &bytes.Buffer{},
	})
	err := i.Close()

	if got != 1 {
		t.Errorf("Close() called %d times; want 1", got)
	}
	if err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
}
