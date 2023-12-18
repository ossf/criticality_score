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
	"fmt"
	"os"
)

func (s *runState) Save() error {
	if s.filename == "" {
		return fmt.Errorf("run state cleared")
	}

	// Temporary filename is based on the destination filename to ensure the
	// temp file is placed on the same volume as the destination file. This
	// avoids potential errors when calling os.Rename.
	tmp, err := os.CreateTemp("", fmt.Sprintf("%s-tmp-*", s.filename))
	if err != nil {
		return fmt.Errorf("temp file create: %w", err)
	}
	tmpFilename := tmp.Name()
	defer func() {
		// tmp is set to nil if close is called eariler. This avoids attempting
		// to close the temporary file more than once.
		if tmp != nil {
			tmp.Close()
		}
		// tmpFilename is set to the emptry string after the rename to prevent
		// remove being called. It is possible that another temp file with the
		// same name has taken its place by the time this has been called.
		if tmpFilename != "" {
			os.Remove(tmpFilename)
		}
	}()

	e := json.NewEncoder(tmp)
	if err := e.Encode(s); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	// Ensure the file's bytes have been written to storage and aren't in a cache.
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("temp file sync: %w", err)
	}

	err := tmp.Close()
	// Clear tmp to prevent the defer func from calling Close.
	tmp = nil
	if err != nil {
		return fmt.Errorf("temp file closing: %w", err)
	}

	if err := os.Rename(tmpFilename, s.filename); err != nil {
		return fmt.Errorf("rename temp %s -> %s: %w", tmpFilename, s.filename, err)
	}
	// Clear tmpFilename to prevent the defer func from calling Remove.
	tmpFilename = ""
	return nil
}
