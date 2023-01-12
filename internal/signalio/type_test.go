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

package signalio

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

func TestTypeString(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name       string
		writerType WriterType
		want       string
	}{
		{name: "csv", writerType: WriterTypeCSV, want: "csv"},
		{name: "json", writerType: WriterTypeJSON, want: "json"},
		{name: "unknown", writerType: WriterType(10), want: ""},
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
		writerType WriterType
		want       string
		err        error
	}{
		{name: "csv", writerType: WriterTypeCSV, want: "csv"},
		{name: "json", writerType: WriterTypeJSON, want: "json"},
		{name: "text", writerType: WriterTypeText, want: "text"},
		{name: "unknown", writerType: WriterType(10), want: "", err: ErrorUnknownWriterType},
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
		want  WriterType
		err   error
	}{
		{input: "csv", want: WriterTypeCSV},
		{input: "json", want: WriterTypeJSON},
		{input: "text", want: WriterTypeText},
		{input: "", want: 0, err: ErrorUnknownWriterType},
		{input: "unknown", want: 0, err: ErrorUnknownWriterType},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var got WriterType
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

func TestWriterType_New(t *testing.T) {
	type args struct {
		emptySets []signal.Set
		extra     []string
	}
	tests := []struct { //nolint:govet
		name string
		t    WriterType
		args args
		want any
	}{
		{
			name: "csv",
			t:    WriterTypeCSV,
			args: args{
				emptySets: []signal.Set{},
				extra:     []string{},
			},
			want: &csvWriter{},
		},
		{
			name: "json",
			t:    WriterTypeJSON,
			args: args{
				emptySets: []signal.Set{},
				extra:     []string{},
			},
			want: &jsonWriter{},
		},
		{
			name: "text",
			t:    WriterTypeText,
			args: args{
				emptySets: []signal.Set{},
				extra:     []string{},
			},
			want: &textWriter{},
		},
		{
			name: "unknown",
			t:    WriterType(math.MaxInt),
			args: args{
				emptySets: []signal.Set{},
				extra:     []string{},
			},
			want: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			got := test.t.New(w, test.args.emptySets, test.args.extra...)

			if reflect.TypeOf(got) != reflect.TypeOf(test.want) {
				t.Fatalf("New() == %v, want %v", reflect.TypeOf(got), reflect.TypeOf(test.want))
			}
		})
	}
}
