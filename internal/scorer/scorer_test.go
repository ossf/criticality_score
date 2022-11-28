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

	"github.com/ossf/criticality_score/internal/collector/signal"
	"github.com/ossf/criticality_score/internal/scorer/algorithm"
)

type testAlgo struct{}

func (t testAlgo) Score(record map[string]float64) float64 {
	sum := 0.0
	for _, v := range record {
		sum += v
	}
	return sum
}

func (t testAlgo) Namespace() signal.Namespace {
	return ""
}

// TODO: Add tests for Score()

func TestFromConfig(t *testing.T) {
	tests := []struct { //nolint:govet
		name              string
		scorerName        string
		fileName          string
		valuesForRegistry map[string]algorithm.Factory
		want              *Scorer
		wantErr           bool
	}{
		{
			name:       "Valid",
			scorerName: "Valid",
			fileName:   "testdata/valid_scorer.yml",
			valuesForRegistry: map[string]algorithm.Factory{
				"linear": func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
					return testAlgo{}, nil
				},
			},
			want: &Scorer{
				name: "Valid",
				a:    testAlgo{},
			},
		},
		{
			name:       "Not an algorithm",
			scorerName: "Not an algorithm",
			fileName:   "testdata/valid_scorer.yml",
			valuesForRegistry: map[string]algorithm.Factory{
				"invalid": func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
					return testAlgo{}, nil
				},
			},
			wantErr: true,
		},
		{
			name:       "Invalid algorithm",
			scorerName: "Invalid algorithm",
			fileName:   "testdata/invalid_scorer.yml",
			valuesForRegistry: map[string]algorithm.Factory{
				"linear": func(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
					return testAlgo{}, nil
				},
			},
			wantErr: true,
		},
		{
			name:              "empty",
			scorerName:        "",
			fileName:          "testdata/valid_scorer.yml",
			valuesForRegistry: map[string]algorithm.Factory{},
			wantErr:           true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			algorithm.GlobalRegistry = algorithm.NewRegistry()

			for k, v := range test.valuesForRegistry {
				algorithm.Register(k, v)
			}

			r, err := os.Open(test.fileName)
			if err != nil {
				t.Fatalf("open file: %v", err)
			}
			got, err := FromConfig(test.scorerName, r)
			if (err != nil) != test.wantErr {
				t.Fatalf("FromConfig() error = %v, wantErr %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("FromConfig() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestScorer_ScoreRaw(t *testing.T) {
	tests := []struct { //nolint:govet
		name string
		s    *Scorer
		raw  map[string]string
		want float64
	}{
		{
			name: "average test",
			s: &Scorer{
				name: "Valid",
				a:    testAlgo{},
			},
			raw: map[string]string{
				"one": "1",
				"two": "2",
			},
			want: 3,
		},
		{
			name: "invalid",
			s: &Scorer{
				name: "invalid",
				a:    testAlgo{},
			},
			raw: map[string]string{
				"invalid number": "abcd",
			},
			want: 0,
		},
		{
			name: "empty",

			s: &Scorer{
				name: "Valid",
				a:    testAlgo{},
			},
			raw:  map[string]string{},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.ScoreRaw(tt.raw); got != tt.want {
				t.Errorf("ScoreRaw() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNameFromFilepath(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     string
	}{
		{
			name:     "default",
			filepath: "z/default123>>.random",
			want:     "default123___score",
		},
		{
			name:     "empty",
			filepath: "",
			want:     "_score",
		},
		{
			name:     "without extension",
			filepath: "z/default123>>",
			want:     "default123___score",
		},
		{
			name:     "only alphanumeric",
			filepath: "z/default123",
			want:     "default123_score",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NameFromFilepath(test.filepath); got != test.want {
				t.Errorf("NameFromFilepath() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestScorer_Name(t *testing.T) {
	test := struct {
		name string
		s    *Scorer
		want string
	}{
		name: "default",
		s: &Scorer{
			name: "Valid",
			a:    testAlgo{},
		},
		want: "Valid",
	}

	if got := test.s.Name(); got != test.want {
		t.Errorf("Name() = %v, want %v", got, test.want)
	}
}
