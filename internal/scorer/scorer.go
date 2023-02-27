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
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/ossf/criticality_score/internal/collector/signal"
	"github.com/ossf/criticality_score/internal/scorer/algorithm"
	_ "github.com/ossf/criticality_score/internal/scorer/algorithm/wam"
)

var ErrEmptyName = fmt.Errorf("name must be non-empty")

type Scorer struct {
	a    algorithm.Algorithm
	name string
}

func FromConfig(name string, r io.Reader) (*Scorer, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	cfg, err := LoadConfig(r)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	a, err := cfg.Algorithm()
	if err != nil {
		return nil, fmt.Errorf("create algorithm: %w", err)
	}
	return &Scorer{
		name: name,
		a:    a,
	}, nil
}

func (s *Scorer) Score(signals []signal.Set) float64 {
	record := make(map[string]float64)
	for _, s := range signals {
		// Get all the signal data from the set change it to a float.
		for k, v := range signal.SetAsMap(s, true) {
			fmt.Println(k, v)
			switch r := v.(type) {
			case float64:
				record[k] = r
			case float32:
				record[k] = float64(r)
			case int:
				record[k] = float64(r)
			case int16:
				record[k] = float64(r)
			case int32:
				record[k] = float64(r)
			case int64:
				record[k] = float64(r)
			case uint:
				record[k] = float64(r)
			case uint16:
				record[k] = float64(r)
			case uint32:
				record[k] = float64(r)
			case uint64:
				record[k] = float64(r)
			case byte:
				record[k] = float64(r)
			}
		}
	}
	return s.a.Score(record)
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

func (s *Scorer) Name() string {
	return s.name
}

func NameFromFilepath(filepath string) string {
	// Get the name of the file used, without the path
	f := path.Base(filepath)

	modifier := func(r rune) rune {
		// Change any non-alphanumeric character into an underscore
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return '_'
		}
		// Convert any characters to lowercase
		return unicode.ToLower(r)
	}

	// Strip the extension
	ext := path.Ext(f)
	f = strings.TrimSuffix(f, ext)

	// Append "_score" to the end
	return strings.Map(modifier, f) + "_score"
}
