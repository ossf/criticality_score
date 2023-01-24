// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package algorithm_test

import (
	"reflect"
	"testing"

	"github.com/ossf/criticality_score/internal/scorer/algorithm"
)

type testAlgo struct{}

func (t testAlgo) Score(record map[string]float64) float64 {
	return 0
}

func TestNewAlgorithm(t *testing.T) {
	// Setup for all tests
	algorithm.Register("test", func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
		return testAlgo{}, nil
	})

	type args struct {
		name   string
		inputs []*algorithm.Input
	}
	tests := []struct { //nolint:govet
		name    string
		args    args
		want    algorithm.Algorithm
		wantErr bool
	}{
		{
			name: "valid registry",

			args: args{
				name:   "test",
				inputs: []*algorithm.Input{},
			},

			want: testAlgo{},
		},
		{
			name: "invalid registry",

			args: args{
				name:   "invalid",
				inputs: []*algorithm.Input{},
			},

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := algorithm.NewAlgorithm(tt.args.name, tt.args.inputs)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewAlgorithm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAlgorithm() got = %v, want %v", got, tt.want)
			}
		})
	}
	t.Cleanup(func() {
		algorithm.GlobalRegistry = algorithm.NewRegistry()
		// Have to do this because the registry is global, and we don't want to
		// pollute it with the test values.
		// Can't create a new registry for every test because the NewAlgorithm
		// function uses GlobalRegistry.
	})
}
