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
	"bytes"
	"errors"
	"net/url"
	"path"
	"strings"
)

type Type int

const (
	TypeFull = Type(iota)
	TypeFile
	TypeDir
)

var ErrorUnknownType = errors.New("unknown marker type")

// String implements the fmt.Stringer interface.
func (t Type) String() string {
	text, err := t.MarshalText()
	if err != nil {
		return ""
	}
	return string(text)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (t Type) MarshalText() ([]byte, error) {
	switch t {
	case TypeFull:
		return []byte("full"), nil
	case TypeFile:
		return []byte("file"), nil
	case TypeDir:
		return []byte("dir"), nil
	default:
		return []byte{}, ErrorUnknownType
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (t *Type) UnmarshalText(text []byte) error {
	switch {
	case bytes.Equal(text, []byte("full")):
		*t = TypeFull
	case bytes.Equal(text, []byte("file")):
		*t = TypeFile
	case bytes.Equal(text, []byte("dir")):
		*t = TypeDir
	default:
		return ErrorUnknownType
	}
	return nil
}

func (t Type) transform(p string) string {
	if t == TypeFull {
		return p
	}
	// Is this a bucket URL?
	if u, err := url.Parse(p); err == nil && u.IsAbs() {
		if u.Scheme == "file" && u.Host == "" {
			p = u.Path
		} else {
			p = strings.TrimPrefix(u.Path, "/")
		}
	}
	if t == TypeDir {
		return path.Dir(p)
	}
	return p
}
