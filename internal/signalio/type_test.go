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

package signalio_test

import (
	"errors"
	"testing"

	"github.com/ossf/criticality_score/internal/signalio"
)

func TestTypeString(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name       string
		writerType signalio.WriterType
		want       string
	}{
		{name: "csv", writerType: signalio.WriterTypeCSV, want: "csv"},
		{name: "json", writerType: signalio.WriterTypeJSON, want: "json"},
		{name: "unknown", writerType: signalio.WriterType(10), want: ""},
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
		writerType signalio.WriterType
		want       string
		err        error
	}{
		{name: "csv", writerType: signalio.WriterTypeCSV, want: "csv"},
		{name: "json", writerType: signalio.WriterTypeJSON, want: "json"},
		{name: "unknown", writerType: signalio.WriterType(10), want: "", err: signalio.ErrorUnknownWriterType},
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
		want  signalio.WriterType
		err   error
	}{
		{input: "csv", want: signalio.WriterTypeCSV},
		{input: "json", want: signalio.WriterTypeJSON},
		{input: "", want: 0, err: signalio.ErrorUnknownWriterType},
		{input: "unknown", want: 0, err: signalio.ErrorUnknownWriterType},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var got signalio.WriterType
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
