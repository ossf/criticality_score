// Copyright 2023 Criticality Score Authors
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

package log

import (
	"bytes"
	"math"
	"testing"
)

func TestLookupEnv(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Env
	}{
		{"dev", "dev", DevEnv},
		{"gcp", "gcp", GCPEnv},
		{"unknown", "unknown", UnknownEnv},
		{"empty", "", UnknownEnv},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := LookupEnv(test.text); got != test.want {
				t.Errorf("LookupEnv() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEnv_String(t *testing.T) {
	tests := []struct { //nolint:govet
		name string
		e    Env
		want string
	}{
		{"dev", DevEnv, "dev"},
		{"gcp", GCPEnv, "gcp"},
		{"unknown", UnknownEnv, "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.e.String(); got != test.want {
				t.Errorf("String() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEnv_UnmarshalText(t *testing.T) {
	tests := []struct { //nolint:govet
		name    string
		e       Env
		text    []byte
		value   string
		wantErr bool
	}{
		{
			name:    "unknown",
			text:    []byte("unknown"),
			value:   "unknown",
			wantErr: true,
		},
		{
			name:  "dev",
			text:  []byte("dev"),
			value: "dev",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.e.UnmarshalText(test.text); (err != nil) != test.wantErr || test.e.String() != test.value {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestEnv_MarshalText(t *testing.T) {
	tests := []struct { //nolint:govet
		name string
		e    Env
		want []byte
	}{
		{"dev", DevEnv, []byte("dev")},
		{"unknown", UnknownEnv, []byte("unknown")},
		{"empty", math.MaxInt, []byte("unknown")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.e.MarshalText()

			if !bytes.Equal(got, test.want) || err != nil {
				// this function never returns an error so err should always be nil
				t.Errorf("MarshalText() got = %v, want %v", got, test.want)
			}
		})
	}
}
