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

//go:build unix || (js && wasm) || wasip1

package localworker

import (
	"encoding/json"
	"fmt"

	"github.com/google/renameio/v2"
)

func (s *runState) Save() error {
	if s.filename == "" {
		return fmt.Errorf("run state cleared")
	}

	pf, err := renameio.NewPendingFile(s.filename)
	if err != nil {
		return fmt.Errorf("pending file: %w", err)
	}
	defer pf.Cleanup()

	e := json.NewEncoder(pf)
	if err := e.Encode(s); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	if err := pf.CloseAtomicallyReplace(); err != nil {
		return fmt.Errorf("atomic close: %w", err)
	}
	return nil
}
