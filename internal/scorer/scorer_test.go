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
	"testing"

	"github.com/ossf/criticality_score/v2/internal/collector/signal"
	"github.com/ossf/criticality_score/v2/internal/scorer/algorithm"
)

type testAlgo struct {
	UpdatedCount signal.Field[int] `signal:"legacy"`
}

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
			name:     "empty",
			filepath: "",
			want:     "_score",
		},
		{
			name:     "without extension",
			filepath: "test",
			want:     "test_score",
		},
		{
			name:     "with extension",
			filepath: "test.json",
			want:     "test_score",
		},
		{
			name:     "with path",
			filepath: "path/to/test.json",
			want:     "test_score",
		},
		{
			name:     "invalid characters",
			filepath: "configuración-+=_básica.yaml",
			want:     "configuración____básica_score",
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

func TestScorer_Score(t *testing.T) {
	type fields struct {
		a    algorithm.Algorithm
		name string
	}
	test := struct {
		name    string
		fields  fields
		signals []signal.Set
		want    float64
	}{
		name: "average test",
		fields: fields{
			a:    testAlgo{},
			name: "Valid",
		},
		signals: []signal.Set{
			&testAlgo{
				UpdatedCount: signal.Val(1),
			},
		},
		want: 1,
	}

	s := &Scorer{
		a:    test.fields.a,
		name: test.fields.name,
	}
	if got := s.Score(test.signals); got != test.want {
		t.Errorf("Score() = %v, want %v", got, test.want)
	}
}
