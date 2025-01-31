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

type testAlgo struct{}

func (t testAlgo) Score(record map[string]float64) float64 {
	return 0
}

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
		name          string
		args          args
		want          *Config
		wantErr       bool
		wantReaderErr bool
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
		{
			name: "want reader error",
			args: args{
				r: "testdata/invalid_config.yaml",
			},
			want:          &Config{},
			wantErr:       true,
			wantReaderErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// read file

			f, err := os.Open(test.args.r)
			if err != nil {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantReaderErr {
				f.Close()
				f = nil
			}

			got, err := LoadConfig(f)
			if (err != nil) != test.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				return
			}

			if !cmp.Equal(got.Inputs, test.want.Inputs) || got.Name != test.want.Name {
				t.Log(cmp.Diff(got.Inputs, test.want.Inputs))
				t.Errorf("LoadConfig() got = %v, want %v", got, test.want)
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
		want    bool
		wantErr bool
		input   map[string]float64
	}{
		{
			name: "invalid condition",
			args: args{
				c: &Condition{},
			},
			wantErr: true,
		},
		{
			name: "c.Not is not nil and returns an error",
			args: args{
				c: &Condition{
					Not: &Condition{},
				},
			},
			wantErr: true,
		},
		{
			name: "c.Not is not nil and c.FieldExists is not empty",
			args: args{
				c: &Condition{
					FieldExists: "test",
					Not:         &Condition{},
				},
			},
			wantErr: true,
		},
		{
			name: "c.Not is not nil and returns a condition",
			args: args{
				c: &Condition{
					Not: &Condition{
						FieldExists: "test",
					},
				},
			},
			input: map[string]float64{"test": 1},
			want:  false,

			wantErr: false,
		},
		{
			name: "c.Not is not nil and returns a condition",
			args: args{
				c: &Condition{
					Not: &Condition{
						FieldExists: "foo",
					},
				},
			},
			input: map[string]float64{"test": 1},
			want:  true,

			wantErr: false,
		},

		// Can't test the c.Not condition because algorithm.Condition is a func and can't be compared
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildCondition(test.args.c)

			if (err != nil) != test.wantErr {
				t.Errorf("buildCondition() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if got == nil && test.wantErr {
				return
			}
			if got(test.input) != test.want {
				t.Errorf("buildCondition() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestConfig_Algorithm(t *testing.T) {
	type fields struct {
		Name   string
		Inputs []*Input
	}
	tests := []struct { //nolint:govet
		name              string
		fields            fields
		valuesForRegistry map[string]algorithm.Factory
		want              algorithm.Algorithm
		wantErr           bool
	}{
		{
			name: "regular",
			fields: fields{
				Name: "test",
				Inputs: []*Input{
					{
						Field:        "test",
						Distribution: "linear",
						Weight:       1,
					},
					{
						Field:        "test2",
						Distribution: "linear",
						Weight:       3,
					},
				},
			},
			valuesForRegistry: map[string]algorithm.Factory{
				"test": func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
					return testAlgo{}, nil
				},
			},
			want:    testAlgo{},
			wantErr: false,
		},
		{
			// This test is for when we call ToAlgorithmInput() and it returns an error.
			// We get the error from ToAlgorithmInput() because Distribution is not a valid
			// distribution when calling algorithm.LookupDistribution()

			name: "invalid",
			fields: fields{
				Name: "test",
				Inputs: []*Input{
					{
						Field:        "test",
						Distribution: "invalid",
						Weight:       1,
					},
				},
			},
			valuesForRegistry: map[string]algorithm.Factory{
				"test": func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
					return testAlgo{}, nil
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			algorithm.GlobalRegistry = algorithm.NewRegistry()

			for name, factory := range test.valuesForRegistry {
				algorithm.Register(name, factory)
			}

			c := &Config{
				Name:   test.fields.Name,
				Inputs: test.fields.Inputs,
			}
			got, err := c.Algorithm()
			if (err != nil) != test.wantErr {
				t.Errorf("Algorithm() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Algorithm() got = %v, want %v", got, test.want)
			}
		})
	}
}
