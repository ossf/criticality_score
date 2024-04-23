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

//go:build !windows

package marker_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ossf/criticality_score/v2/cmd/enumerate_github/marker"
)

func TestWrite(t *testing.T) {
	want := "this/is/a/path"
	dir := t.TempDir()
	file := path.Join(dir, "marker.test")

	err := marker.Write(context.Background(), marker.TypeFull, file, want)
	if err != nil {
		t.Fatalf("Write() = %v, want no error", err)
	}

	markerContents, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile() = %v, want no error", err)
	}
	if got := strings.TrimSpace(string(markerContents)); got != want {
		t.Fatalf("marker contents = %q, want %q", got, want)
	}
}
