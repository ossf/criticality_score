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

package cloudstorage

import (
	"context"
	"net/url"
	"testing"
)

func TestParseBucketAndPrefixAbsLocalFile(t *testing.T) {
	b, p, err := parseBucketAndPrefix("/path/to/file")
	if err != nil {
		t.Fatalf("parseBucketAndPrefix() = %v, want no error", err)
	}
	assertBucket(t, b, fileScheme, "", "/path/to/", map[string]string{"metadata": "skip"})
	if p != "file" {
		t.Fatalf("Prefix = %v; want file", p)
	}
}

func TestParseBucketAndPrefixRelativeLocalFile(t *testing.T) {
	b, p, err := parseBucketAndPrefix("path/to/file")
	if err != nil {
		t.Fatalf("parseBucketAndPrefix() = %v, want no error", err)
	}
	assertBucket(t, b, fileScheme, ".", "/path/to/", map[string]string{"metadata": "skip"})
	if p != "file" {
		t.Fatalf("Prefix = %v; want file", p)
	}
}

func TestParseBucketAndPrefixS3URL(t *testing.T) {
	b, p, err := parseBucketAndPrefix("s3://bucket/path/to/file")
	if err != nil {
		t.Fatalf("parseBucketAndPrefix() = %v, want no error", err)
	}
	assertBucket(t, b, "s3", "bucket", "", map[string]string{})
	if p != "path/to/file" {
		t.Fatalf("Prefix = %v; want path/to/file", p)
	}
}

func TestNewWriterNoScheme(t *testing.T) {
	_, _, err := parseBucketAndPrefix("//example.com/path/to/file")
	if err == nil {
		t.Fatal("parseBucketAndPrefix() = nil; want an error")
	}
}

func TestNewWriterInvalidURL(t *testing.T) {
	_, err := NewWriter(context.Background(), "::")
	if err == nil {
		t.Fatal("NewWriter() = nil; want an error")
	}
}

func TestNewWriterUnsupportedScheme(t *testing.T) {
	_, err := NewWriter(context.Background(), "ftp://bucket/path/to/file")
	if err == nil {
		t.Fatal("NewWriter() = nil; want an error")
	}
}

func TestNewWriter(t *testing.T) {
	w, err := NewWriter(context.Background(), "mem://bucket/path/to/file")
	if err != nil {
		t.Fatalf("NewWriter() = %v, want no error", err)
	}
	if w == nil {
		t.Fatal("NewWriter() = nil, want a writer")
	}
}

func assertBucket(t *testing.T, bucket, wantScheme, wantHost, wantPath string, wantQuery map[string]string) {
	t.Helper()
	u, err := url.Parse(bucket)
	if err != nil {
		t.Fatalf("Bucket is not a valid url: %v", err)
	}
	if u.Scheme != wantScheme {
		t.Errorf("Bucket scheme = %q, want %q", u.Scheme, wantScheme)
	}
	if u.Host != wantHost {
		t.Errorf("Bucket host = %q, want %q", u.Host, wantHost)
	}
	if u.Path != wantPath {
		t.Errorf("Bucket path = %q, want %q", u.Path, wantPath)
	}
	for k, want := range wantQuery {
		if !u.Query().Has(k) {
			t.Errorf("Bucket query has no key %q", k)
		}
		if got := u.Query().Get(k); got != want {
			t.Errorf("Bucket query %q = %q, want %q", k, got, want)
		}
	}
}
