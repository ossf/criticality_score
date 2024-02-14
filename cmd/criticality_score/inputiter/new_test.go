// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inputiter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ossf/criticality_score/cmd/criticality_score/inputiter"
)

func TestNew_SingleURL(t *testing.T) {
	want := "https://github.com/ossf/criticality_score"
	i, err := inputiter.New([]string{want})
	if err != nil {
		t.Fatalf("New() = %#v; want no error", err)
	}
	defer i.Close()

	// Move to the first item
	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}

	// Get the single item
	got := i.Item()
	if got != want {
		t.Errorf("Item() = %v; want %v", got, want)
	}

	// Ensure the iterator is now empty
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
	}
}

func TestNew_MultipleURL(t *testing.T) {
	want := []string{
		"https://github.com/ossf/criticality_score",
		"https://github.com/ossf/scorecard",
	}
	i, err := inputiter.New(want)
	if err != nil {
		t.Fatalf("New() = %v; want no error", err)
	}
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

func TestNew_MissingFileIsURL(t *testing.T) {
	want := "this/is/a/file/that/doesnt/exists"
	i, err := inputiter.New([]string{want})
	if err != nil {
		t.Fatalf("New() = %v; want no error", err)
	}
	defer i.Close()

	// Move to the first item
	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err() = %v; want no error", err)
	}

	// Get the single item
	got := i.Item()
	if got != want {
		t.Errorf("Item() = %v; want %v", got, want)
	}

	// Ensure the iterator is now empty
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
	if err := i.Err(); err != nil {
		t.Errorf("Err()#2 = %v; want no error", err)
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

	i, err := inputiter.New([]string{path})
	if err != nil {
		t.Fatalf("New() = %v; want no error", err)
	}
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
	if err == nil {
		t.Errorf("New() = %#v; want error", err)
	}
	if err == nil {
		defer i.Close()
	}
}
