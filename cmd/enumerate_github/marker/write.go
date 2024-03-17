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

package marker

import (
	"context"
	"fmt"

	"github.com/ossf/criticality_score/v2/internal/cloudstorage"
)

func Write(ctx context.Context, t Type, markerFile, outFile string) (err error) {
	marker, e := cloudstorage.NewWriter(ctx, markerFile)
	if e != nil {
		return fmt.Errorf("open marker: %w", e)
	}
	defer func() {
		if e := marker.Close(); e != nil {
			// Return the error using the named return value if it isn't
			// already set.
			if e == nil {
				err = fmt.Errorf("closing marker: %w", e)
			}
		}
	}()
	if _, e := fmt.Fprintln(marker, t.transform(outFile)); e != nil {
		err = fmt.Errorf("writing marker: %w", e)
	}
	return
}
