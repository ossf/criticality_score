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
	"fmt"
	"io"
	"strconv"

	"github.com/ossf/criticality_score/internal/scorer/algorithm"
	_ "github.com/ossf/criticality_score/internal/scorer/algorithm/wam"
)

type Scorer struct {
	a algorithm.Algorithm
}

func FromConfig(r io.Reader) (*Scorer, error) {
	cfg, err := LoadConfig(r)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	a, err := cfg.Algorithm()
	if err != nil {
		return nil, fmt.Errorf("create algorithm: %w", err)
	}
	return &Scorer{
		a: a,
	}, nil
}

func (s *Scorer) ScoreRaw(raw map[string]string) float64 {
	record := make(map[string]float64)
	for k, rawV := range raw {
		// TODO: improve this behavior
		v, err := strconv.ParseFloat(rawV, 64)
		if err != nil {
			// Failed to parse raw into a float, ignore the field
			continue
		}
		record[k] = v
	}
	return s.a.Score(record)
}
