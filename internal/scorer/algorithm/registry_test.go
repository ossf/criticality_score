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

package algorithm

import (
	"reflect"
	"testing"
)

type testAlgo struct{}

func (t testAlgo) Score(record map[string]float64) float64 {
	return 0
}

func TestNewAlgorithm(t *testing.T) {
	type args struct {
		name   string
		inputs []*Input
	}
	tests := []struct { //nolint:govet
		name              string
		args              args
		valuesForRegistry map[string]Factory
		want              Algorithm
		wantErr           bool
	}{
		{
			name: "valid registry",

			args: args{
				name:   "test",
				inputs: []*Input{},
			},

			valuesForRegistry: map[string]Factory{
				"test": func(inputs []*Input) (Algorithm, error) {
					return testAlgo{}, nil
				},
			},

			want: testAlgo{},
		},
		{
			name: "invalid registry",

			args: args{
				name:   "invalid",
				inputs: []*Input{},
			},

			valuesForRegistry: map[string]Factory{
				"test": func(inputs []*Input) (Algorithm, error) {
					return testAlgo{}, nil
				},
			},

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.valuesForRegistry {
				Register(k, v)
			}

			got, err := NewAlgorithm(tt.args.name, tt.args.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAlgorithm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAlgorithm() got = %v, want %v", got, tt.want)
			}
		})
	}
}
