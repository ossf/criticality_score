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

package algorithm

import "fmt"

// GlobalRegistry is the global, application wide, registry for all algorithms.
var GlobalRegistry = NewRegistry()

// Registry is used to map a name to a Factory that creates Algorithm instances
// for the given name.
type Registry struct {
	as map[string]Factory
}

// NewRegistry returns a new Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		as: make(map[string]Factory),
	}
}

// Register adds the Factory for the corresponding Key to the registry.
//
// If another Factory has been registered with the same Key it will be
// replaced.
func (r *Registry) Register(name string, f Factory) {
	r.as[name] = f
}

// NewAlgorithm generates a new instance of Algorithm for the supplied name and
// fields.
//
// If the registry does not have a Factory for the supplied name an error will
// be returned.
//
// If the Algorithm fails to be created by the Factory, an error will also be
// returned and the Algorithm will be nil.
func (r *Registry) NewAlgorithm(name string, inputs []*Input) (Algorithm, error) {
	f, ok := r.as[name]
	if !ok {
		return nil, fmt.Errorf("unknown algorithm %s", name)
	}
	return f(inputs)
}

// Register calls Register on the GlobalRegistry.
func Register(name string, f Factory) {
	GlobalRegistry.Register(name, f)
}

// NewAlgorithm calls NewAlgorithm on the GlobalRegsitry.
func NewAlgorithm(name string, inputs []*Input) (Algorithm, error) {
	return GlobalRegistry.NewAlgorithm(name, inputs)
}
