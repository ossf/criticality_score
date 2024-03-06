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

package scorer

import (
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ossf/criticality_score/v2/internal/scorer/algorithm"
)

func TestInput_ToAlgorithmInput(t *testing.T) {
	type fields struct {
		Bounds       *algorithm.Bounds
		Condition    *Condition
		Field        string
		Distribution string
		Tags         []string
		Weight       float64
	}
	//nolint:govet
	tests := []struct {
		name    string
		fields  fields
		want    *algorithm.Input
		wantErr bool
	}{
		{
			name: "unknown distribution error",
			fields: fields{
				Field: "test",
			},
			want:    &algorithm.Input{},
			wantErr: true,
		},
		{
			name: "distribution value set",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
			},
			want: &algorithm.Input{
				Source:       algorithm.Value(algorithm.Field("test")),
				Distribution: algorithm.LookupDistribution("linear"),
			},
			wantErr: false,
		},
		{
			name: "valid condition",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
				Condition: &Condition{
					FieldExists: "test",
				},
			},
			want: &algorithm.Input{
				Distribution: algorithm.LookupDistribution("linear"),
			},
			wantErr: false,
		},
		{
			name: "invalid condition",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
				Condition: &Condition{
					FieldExists: "test",
					Not: &Condition{
						FieldExists: "test",
					},
				},
			},
			want:    &algorithm.Input{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			i := &Input{
				Bounds:       test.fields.Bounds,
				Condition:    test.fields.Condition,
				Field:        test.fields.Field,
				Distribution: test.fields.Distribution,
				Tags:         test.fields.Tags,
				Weight:       test.fields.Weight,
			}
			got, err := i.ToAlgorithmInput()
			if (err != nil) != test.wantErr {
				t.Fatalf("ToAlgorithmInput() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			// Comparing specific fields independently because some of the structs have a func as a field which
			// can't be used for comparison with reflect.DeepEqual()

			if got.Distribution.String() != test.want.Distribution.String() {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, test.want)
			}
			if got.Bounds != test.want.Bounds {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, test.want)
			}
			if got.Weight != test.want.Weight {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, test.want)
			}
			if !reflect.DeepEqual(got.Tags, test.want.Tags) {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	type args struct {
		r string
	}
	//nolint:govet
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "valid config",
			args: args{
				r: "testdata/default_config.yml",
			},
			want: &Config{
				Name: "test",
				Inputs: []*Input{
					{
						Field:        "test",
						Distribution: "linear",
						Weight:       1,
					},
				},
			},
		},
		{
			name: "invalid yaml",
			args: args{
				r: "testdata/invalid_config.yaml",
			},
			want:    &Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// read file

			f, err := os.Open(tt.args.r)
			if err != nil {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := LoadConfig(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !cmp.Equal(got.Inputs, tt.want.Inputs) || got.Name != tt.want.Name {
				t.Log(cmp.Diff(got.Inputs, tt.want.Inputs))
				t.Errorf("LoadConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildCondition(t *testing.T) {
	type args struct {
		c *Condition
	}
	//nolint:govet
	tests := []struct {
		name    string
		args    args
		want    algorithm.Condition
		wantErr bool
	}{
		{
			name: "invalid condition",
			args: args{
				c: &Condition{},
			},
			want:    nil,
			wantErr: true,
		},

		// Can't test the c.Not condition because algorithm.Condition is a func and can't be compared
	}
	test := tests[0]
	got, err := buildCondition(test.args.c)

	if (err != nil) != test.wantErr {
		t.Errorf("buildCondition() error = %v, wantErr %v", err, test.wantErr)
		return
	}
	if !reflect.DeepEqual(got, test.want) {
		t.Errorf("buildCondition() got = %v, want %v", got, test.want)
	}
}
