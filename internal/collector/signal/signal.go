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

package signal

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
)

type Namespace string

func (ns Namespace) String() string {
	return string(ns)
}

const (
	// nameSeparator is used to separate the namespace from the field name.
	nameSeparator = '.'

	// NamespaceLegacy is an internal namespace used for fields that provide
	// compatibility with the legacy python implementation.
	NamespaceLegacy Namespace = "legacy"

	// The following constants are used while parsing the tags for the struct
	// fields in a Set.
	fieldTagName      = "signal"
	fieldTagIgnore    = "-"
	fieldTagLegacy    = "legacy"
	fieldTagSeparator = ","
)

const (
	NamespaceRepo   Namespace = "repo"
	NamespaceIssues Namespace = "issues"
)

var (
	// validName is used to validate namespace and field names.
	validName = regexp.MustCompile("^[a-z0-9_]+$")

	// valuerType caches the reflect.Type representation of the valuer
	// interface.
	valuerType = reflect.TypeOf((*valuer)(nil)).Elem()
)

type SupportedType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string | time.Time
}

// valuer is provides access to the field's value without needing to use
// the concrete generic. It is also a marker interface used to identify the
// Signal field in a struct.
type valuer interface {
	Value() any
}

type Field[T SupportedType] struct {
	value T
	set   bool
}

func (s *Field[T]) Set(v T) {
	s.value = v
	s.set = true
}

func (s *Field[T]) Get() T {
	if !s.set {
		var zero T
		return zero
	}
	return s.value
}

func (s *Field[T]) IsSet() bool {
	return s.set
}

func (s *Field[T]) Unset() {
	s.set = false
}

func (s Field[T]) Value() any {
	if !s.set {
		return nil
	} else {
		return s.value
	}
}

// Val is used to create a Field instance that is already set with the value v.
//
// This method is particularly useful when creating an new instance of a Set.
//
//	    s = &FooSet{
//		       myField: signal.Val("hello, world!")
//	    }
func Val[T SupportedType](v T) Field[T] {
	var f Field[T]
	f.Set(v)
	return f
}

type Set interface {
	Namespace() Namespace
}

type fieldConfig struct {
	name   string
	legacy bool
}

// ValidateSet tests whether a Set is valid.
//
// An invalid Set will return an error with the first test that failed. A valid
// Set will return nil.
func ValidateSet(s Set) error {
	if ns := string(s.Namespace()); !validName.MatchString(ns) {
		return fmt.Errorf("namespace '%s' contains invalid characters", ns)
	}
	cb := func(f *fieldConfig, _ any) error {
		if !validName.MatchString(f.name) {
			return fmt.Errorf("field name '%s' contains invalid character", f.name)
		}
		return nil
	}
	return iterSetFields(s, cb)
}

// parseStructField processes a field sf in a Set to extract the field's
// configuration from the field's tag.
//
// If the field is not of type Field, or is configured to be ignored, this
// method will return nil.
func parseStructField(sf reflect.StructField) *fieldConfig {
	if !sf.Type.Implements(valuerType) {
		// Ignore fields that don't implement the valuer interface as
		// these are not Field and should be skipped.
		return nil
	}
	tag := sf.Tag.Get(fieldTagName)
	if tag == fieldTagIgnore {
		// The field is marked to be ignored, so skip it.
		return nil
	}
	f := &fieldConfig{
		name:   strcase.ToSnake(sf.Name),
		legacy: false,
	}
	if tag != "" {
		parts := strings.Split(tag, fieldTagSeparator)
		for _, p := range parts {
			switch strings.TrimSpace(p) {
			case "":
				// noop
			case fieldTagLegacy:
				f.legacy = true
			default:
				f.name = p
			}
		}
	}
	return f
}

// iterSetFields is an internal helper for looping across all the Fields in s.
// It is also responsible for parsing the struct's tag.
//
// The function cb is called for each field that will be present in the output.
func iterSetFields(s Set, cb func(*fieldConfig, any) error) error {
	vs := reflect.ValueOf(s).Elem()
	tfs := reflect.VisibleFields(reflect.TypeOf(s).Elem())
	for _, sf := range tfs {
		f := parseStructField(sf)
		if f == nil {
			continue
		}
		val := vs.FieldByIndex(sf.Index).Interface().(valuer)
		// Grab the value and call the cb with all the bits
		v := val.Value()
		if err := cb(f, v); err != nil {
			return err
		}
	}
	return nil
}

// SetFields returns a slice containing the names of the fields for s.
//
// If namespace is true the field names will be prefixed with the namespace.
func SetFields(s Set, namespace bool) []string {
	var fs []string
	prefix := ""
	legacyPrefix := ""
	if namespace {
		prefix = fmt.Sprintf("%s%c", s.Namespace(), nameSeparator)
		legacyPrefix = fmt.Sprintf("%s%c", NamespaceLegacy, nameSeparator)
	}
	_ = iterSetFields(s, func(f *fieldConfig, _ any) error {
		if f.legacy {
			fs = append(fs, legacyPrefix+f.name)
		} else {
			fs = append(fs, prefix+f.name)
		}
		return nil
	})
	return fs
}

// SetValues returns a slice containing the values for each field for s.
//
// The values are either `nil` if the Field is not set, or the value that was
// set.
func SetValues(s Set) []any {
	var vs []any
	_ = iterSetFields(s, func(_ *fieldConfig, v any) error {
		vs = append(vs, v)
		return nil
	})
	return vs
}

// SetAsMap returns a map where each field name is mapped to the value of the
// field.
//
// Fields are named as returned by SetFields and values are the same as those
// returned by SetValues.
func SetAsMap(s Set, namespace bool) map[string]any {
	fs := SetFields(s, namespace)
	vs := SetValues(s)
	m := make(map[string]any)
	for i := range fs {
		m[fs[i]] = vs[i]
	}
	return m
}

// SetAsMapWithNamespace returns a map where the outer map contains keys
// corresponding to the namespace, and each inner map contains each field name
// mapped to the value of the field.
func SetAsMapWithNamespace(s Set) map[string]map[string]any {
	m := make(map[string]map[string]any)
	_ = iterSetFields(s, func(f *fieldConfig, v any) error {
		// Determine which namespace to use.
		ns := s.Namespace().String()
		if f.legacy {
			ns = NamespaceLegacy.String()
		}
		innerM, ok := m[ns]
		if !ok {
			innerM = make(map[string]any)
			m[ns] = innerM
		}
		innerM[f.name] = v
		return nil
	})
	return m
}
