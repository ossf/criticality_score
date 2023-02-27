package inputiter_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/ossf/criticality_score/cmd/criticality_score/inputiter"
)

func TestNew_SingleURL(t *testing.T) {
	want := "https://github.com/ossf/criticality_score"
	i, _ := inputiter.New([]string{want})
	defer i.Close()

	// Move to the first item
	if got := i.Next(); !got {
		t.Errorf("Next() = %v, want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v, want no error", err)
	}

	// Get the single item
	got := i.Item()
	if got != want {
		t.Errorf("Item() = %v, want %v", got, want)
	}

	// Ensure the iterator is now empty
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v, want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v, want no error", err)
	}
}

func TestNew_MultipleURL(t *testing.T) {
	want := []string{
		"https://github.com/ossf/criticality_score",
		"https://github.com/ossf/scorecard",
	}
	i, _ := inputiter.New(want)
	defer i.Close()

	var got []string
	for i.Next() {
		item := i.Item()
		got = append(got, item)
	}

	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no err", err)
	}

	if !slices.Equal(got, want) {
		t.Errorf("Iterator return %v; want %v", got, want)
	}
}

func TestNew_MissingFile(t *testing.T) {
	want := "this/is/a/file/that/doesnt/exists"
	i, _ := inputiter.New([]string{want})
	defer i.Close()

	// Move to the first item
	if got := i.Next(); !got {
		t.Errorf("Next() = %v, want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v, want no error", err)
	}

	// Get the single item
	got := i.Item()
	if got != want {
		t.Errorf("Item() = %v, want %v", got, want)
	}

	// Ensure the iterator is now empty
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v, want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v, want no error", err)
	}
}

func TestNew_URLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "urls.txt")
	want := []string{
		"https://github.com/ossf/criticality_score",
		"https://github.com/ossf/scorecard",
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()
	for _, url := range want {
		if _, err := fmt.Fprintln(f, url); err != nil {
			t.Fatalf("Failed to write to test file: %v", err)
		}
	}

	i, _ := inputiter.New([]string{path})
	defer i.Close()

	var got []string
	for i.Next() {
		item := i.Item()
		got = append(got, item)
	}

	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no err", err)
	}

	if !slices.Equal(got, want) {
		t.Errorf("Iterator return %v; want %v", got, want)
	}
}

func TestNew_InvalidURL(t *testing.T) {
	want := ":this.is/not/a/url"
	i, err := inputiter.New([]string{want})
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("New() = %v; want %v", err, os.ErrNotExist)
	}
	if err == nil {
		defer i.Close()
	}
}
