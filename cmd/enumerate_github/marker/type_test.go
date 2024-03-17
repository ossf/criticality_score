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

func TestTransform(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		t    marker.Type
	}{
		{
			name: "full-bucket-gs",
			t:    marker.TypeFull,
			in:   "gs://bucket/path/to/file.txt?arg",
			want: "gs://bucket/path/to/file.txt?arg",
		},
		{
			name: "file-bucket-gs",
			t:    marker.TypeFile,
			in:   "gs://bucket/path/to/file.txt?arg",
			want: "path/to/file.txt",
		},
		{
			name: "dir-bucket-gs",
			t:    marker.TypeDir,
			in:   "gs://bucket/path/to/file.txt?arg",
			want: "path/to",
		},
		{
			name: "file-bucket-file-abs",
			t:    marker.TypeFile,
			in:   "file:///path/to/file.txt?arg",
			want: "/path/to/file.txt",
		},
		{
			name: "file-bucket-file-rel",
			t:    marker.TypeFile,
			in:   "file://./path/to/file.txt?arg",
			want: "path/to/file.txt",
		},
		{
			name: "full-path-abs",
			t:    marker.TypeFull,
			in:   "/path/to/file.txt",
			want: "/path/to/file.txt",
		},
		{
			name: "file-path-abs",
			t:    marker.TypeFile,
			in:   "/path/to/file.txt",
			want: "/path/to/file.txt",
		},
		{
			name: "dir-path-abs",
			t:    marker.TypeDir,
			in:   "/path/to/file.txt",
			want: "/path/to",
		},
		{
			name: "full-path-rel",
			t:    marker.TypeFull,
			in:   "../path/to/file.txt",
			want: "../path/to/file.txt",
		},
		{
			name: "file-path-rel",
			t:    marker.TypeFile,
			in:   "../path/to/file.txt",
			want: "../path/to/file.txt",
		},
		{
			name: "dir-path-rel",
			t:    marker.TypeDir,
			in:   "../path/to/file.txt",
			want: "../path/to",
		},
		{
			name: "full-invalid-url",
			t:    marker.TypeFull,
			in:   "::/path/to/file.txt",
			want: "::/path/to/file.txt",
		},
		{
			name: "file-invalid-url",
			t:    marker.TypeFile,
			in:   "::/path/to/file.txt",
			want: "::/path/to/file.txt",
		},
		{
			name: "dir-invalid-url",
			t:    marker.TypeDir,
			in:   "::/path/to/file.txt",
			want: "::/path/to",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			markerFile := path.Join(t.TempDir(), "marker.out")
			err := marker.Write(context.Background(), test.t, markerFile, test.in)
			if err != nil {
				t.Fatalf("Write() = %v, want no error", err)
			}
			markerContents, err := os.ReadFile(markerFile)
			if err != nil {
				t.Fatalf("ReadFile() = %v, want no error", err)
			}
			if got := strings.TrimSpace(string(markerContents)); got != test.want {
				t.Fatalf("marker content = %q, want %q", got, test.want)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		in      string
		want    marker.Type
		wantErr bool
	}{
		{
			in:   "full",
			want: marker.TypeFull,
		},
		{
			in:   "file",
			want: marker.TypeFile,
		},
		{
			in:   "dir",
			want: marker.TypeDir,
		},
		{
			in:      "notamarker",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			var got marker.Type
			err := got.UnmarshalText([]byte(test.in))
			if test.wantErr && err == nil {
				t.Fatal("UnmarshalText() = nil, want an error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("UnmarshalText() = %v, want no error", err)
			}
			if test.want != got {
				t.Fatalf("UnmarshalText() parsed %s, want %s", got, test.want)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		want    string
		in      marker.Type
		wantErr bool
	}{
		{
			in:   marker.TypeFull,
			want: "full",
		},
		{
			in:   marker.TypeFile,
			want: "file",
		},
		{
			in:   marker.TypeDir,
			want: "dir",
		},
		{
			in:      marker.Type(99999),
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.in.String(), func(t *testing.T) {
			got, err := test.in.MarshalText()
			if test.wantErr && err == nil {
				t.Fatal("UnmarshalText() = nil, want an error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("UnmarshalText() = %v, want no error", err)
			}
			if test.want != string(got) {
				t.Fatalf("UnmarshalText() parsed %s, want %s", got, test.want)
			}
		})
	}
}
