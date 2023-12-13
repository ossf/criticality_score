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

func TestBatchIter_Empty(t *testing.T) {
	var b bytes.Buffer
	i := iterator.Batch(iterator.Lines(io.NopCloser(&b)), 10)

	if got := i.Next(); got {
		t.Errorf("Next() = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
}

func TestBatchIter_SingleLine(t *testing.T) {
	want := []string{"test line"}
	b := bytes.NewBuffer([]byte(strings.Join(want, "\n")))
	i := iterator.Batch(iterator.Lines(io.NopCloser(b)), 10)

	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
	if got := i.Item(); !slices.Equal(got, want) {
		t.Errorf("Item() = %v; want %v", got, want)
	}
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
	}
}

func TestBatchIter_MultiLineSingleBatch(t *testing.T) {
	want := []string{"line one", "line two", "line three"}
	b := bytes.NewBuffer([]byte(strings.Join(want, "\n")))
	i := iterator.Batch(iterator.Lines(io.NopCloser(b)), 10)

	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
	if got := i.Item(); !slices.Equal(got, want) {
		t.Errorf("Item() = %v; want %v", got, want)
	}
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
	}
}

func TestBatchIter_MultiLineMultiBatch(t *testing.T) {
	want1 := []string{"line one", "line two"}
	want2 := []string{"line three"}
	b := bytes.NewBuffer([]byte(strings.Join(append(want1, want2...), "\n")))
	i := iterator.Batch(iterator.Lines(io.NopCloser(b)), 2)

	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
	if got := i.Item(); !slices.Equal(got, want1) {
		t.Errorf("Item()#1 = %v; want %v", got, want1)
	}
	if got := i.Next(); !got {
		t.Errorf("Next()#2 = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
	}
	if got := i.Item(); !slices.Equal(got, want2) {
		t.Errorf("Item()#2 = %v; want %v", got, want2)
	}
	if got := i.Next(); got {
		t.Errorf("Next()#3 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#3 = %v; want no error", err)
	}
}

func TestBatchIter_Error(t *testing.T) {
	want := errors.New("error")
	r := iotest.ErrReader(want)
	i := iterator.Batch(iterator.Lines(io.NopCloser(r)), 10)

	if got := i.Next(); got {
		t.Errorf("Next() = %v; want false", got)
	}
	if err := i.Err(); err == nil || !errors.Is(err, want) {
		t.Errorf("Err() = %v; want %v", err, want)
	}
}

func TestBatchIter_Close(t *testing.T) {
	got := 0
	i := iterator.Batch(iterator.Lines(&struct {
		closerFn
		io.Reader
	}{
		closerFn: closerFn(func() error {
			got++
			return nil
		}),
		Reader: &bytes.Buffer{},
	}), 10)
	err := i.Close()

	if got != 1 {
		t.Errorf("Close() called %d times; want 1", got)
	}
	if err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}
}
