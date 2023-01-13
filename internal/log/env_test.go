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
	"math"
	"reflect"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LookupEnv(tt.text); got != tt.want {
				t.Errorf("LookupEnv() = %v, want %v", got, tt.want)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
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
	for _, tt := range tests {
		t.Setenv(string(tt.text), tt.value)
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.UnmarshalText(tt.text); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.e.MarshalText() // this function never returns an error

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalText() got = %v, want %v", got, tt.want)
			}
		})
	}
}
