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

package localworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type runState struct {
	JobTime  time.Time `json:"job-time"`
	filename string    `json:"-"`
	Attempt  int       `json:"attempt"`
	Shard    int32     `json:"shard"`
}

func loadState(filename string) (*runState, error) {
	f, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return &runState{
			JobTime:  time.Now().UTC(),
			filename: filename,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filename, err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	var s *runState
	if err := d.Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding json: %w", err)
	}
	s.filename = filename
	return s, nil
}

func (s *runState) Clear() error {
	if s.filename == "" {
		return fmt.Errorf("run state cleared")
	}
	f := s.filename
	s.filename = ""

	if err := os.Remove(f); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("removing file: %w", err)
	}
	return nil
}
