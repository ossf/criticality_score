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
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/ossf/criticality_score/internal/scorer/algorithm"
)

type Condition struct {
	Not         *Condition `yaml:"not"`
	FieldExists string     `yaml:"field_exists"`
}

type Input struct {
	Bounds       *algorithm.Bounds `yaml:"bounds"`
	Condition    *Condition        `yaml:"condition"`
	Field        string            `yaml:"field"`
	Distribution string            `yaml:"distribution"`
	Tags         []string          `yaml:"tags"`
	Weight       float64           `yaml:"weight"`
}

// UnmarshalYAML Implements yaml.Unmarshaler interface.
func (i *Input) UnmarshalYAML(value *yaml.Node) error {
	type RawInput Input
	raw := &RawInput{
		Weight:       1,
		Distribution: algorithm.DefaultDistributionName,
	}
	if err := value.Decode(raw); err != nil {
		return err
	}
	if raw.Field == "" {
		return errors.New("field must be set")
	}
	*i = Input(*raw)

	if i.Weight <= 0 {
		return errors.New("weight must be greater than 0")
	}

	return nil
}

func buildCondition(c *Condition) (algorithm.Condition, error) {
	if c.FieldExists != "" && c.Not != nil {
		return nil, errors.New("only one field of condition must be set")
	}
	if c.FieldExists != "" {
		return algorithm.ExistsCondition(algorithm.Field(c.FieldExists)), nil
	}
	if c.Not != nil {
		innerC, err := buildCondition(c.Not)
		if err != nil {
			return nil, err
		}
		return algorithm.NotCondition(innerC), nil
	}
	return nil, errors.New("one condition field must be set")
}

// ToAlgorithmInput returns an instance of algorithm.Input that is constructed.
func (i *Input) ToAlgorithmInput() (*algorithm.Input, error) {
	var v algorithm.Value
	v = algorithm.Field(i.Field)
	if i.Condition != nil {
		c, err := buildCondition(i.Condition)
		if err != nil {
			return nil, err
		}
		v = &algorithm.ConditionalValue{
			Condition: c,
			Inner:     v,
		}
	}
	d := algorithm.LookupDistribution(i.Distribution)
	if d == nil {
		return nil, fmt.Errorf("unknown distribution %s", i.Distribution)
	}
	return &algorithm.Input{
		Bounds:       i.Bounds,
		Weight:       i.Weight,
		Distribution: d,
		Source:       v,
		Tags:         i.Tags,
	}, nil
}

// Config is used to specify an algorithm and its given set of Fields and
// Options.
//
// This structure is used for parsing a YAML file and returning an instance of
// an Algorithm based on the configuration.
type Config struct {
	Name   string   `yaml:"algorithm"`
	Inputs []*Input `yaml:"inputs"`
}

// LoadConfig will parse the YAML data from the reader and return a Config
// that can be used to obtain an Algorithm instance.
//
// If the data cannot be parsed an error will be returned.
func LoadConfig(r io.Reader) (*Config, error) {
	c := &Config{}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}

	return c, nil
}

// Algorithm returns an instance of Algorithm that is constructed from the
// Config.
//
// nil will be returned if the algorithm cannot be returned.
func (c *Config) Algorithm() (algorithm.Algorithm, error) {
	var inputs []*algorithm.Input
	for _, i := range c.Inputs {
		input, err := i.ToAlgorithmInput()
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return algorithm.NewAlgorithm(c.Name, inputs)
}
