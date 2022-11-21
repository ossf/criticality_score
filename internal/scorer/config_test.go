package scorer

import (
	"os"
	"reflect"
	"testing"

	"github.com/ossf/criticality_score/internal/scorer/algorithm"

	"github.com/google/go-cmp/cmp"
)

func TestInput_ToAlgorithmInput(t *testing.T) {
	type fields struct {
		Bounds       *algorithm.Bounds
		Condition    *Condition
		Field        string
		Distribution string
		Tags         []string
		Weight       float64
	}
	tests := []struct {
		name    string
		fields  fields
		want    *algorithm.Input
		wantErr bool
	}{
		{
			name: "unknown distribution error",
			fields: fields{
				Field: "test",
			},
			want:    &algorithm.Input{},
			wantErr: true,
		},
		{
			name: "distribution value set",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
			},
			want: &algorithm.Input{
				Source:       algorithm.Value(algorithm.Field("test")),
				Distribution: algorithm.LookupDistribution("linear"),
			},
			wantErr: false,
		},
		{
			name: "Valid condition",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
				Condition: &Condition{
					FieldExists: "test",
				},
			},
			want: &algorithm.Input{
				Distribution: algorithm.LookupDistribution("linear"),
			},
			wantErr: false,
		},
		{
			name: "Invalid condition",
			fields: fields{
				Field:        "test",
				Distribution: "linear",
				Condition: &Condition{
					FieldExists: "test",
					Not: &Condition{
						FieldExists: "test",
					},
				},
			},
			want:    &algorithm.Input{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Input{
				Bounds:       tt.fields.Bounds,
				Condition:    tt.fields.Condition,
				Field:        tt.fields.Field,
				Distribution: tt.fields.Distribution,
				Tags:         tt.fields.Tags,
				Weight:       tt.fields.Weight,
			}
			got, err := i.ToAlgorithmInput()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToAlgorithmInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Comparing specific fields independently because some of the structs have a func as a field which
			// can't be used for comparison with reflect.DeepEqual()

			if got.Distribution.String() != tt.want.Distribution.String() {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, tt.want)
			}
			if got.Bounds != tt.want.Bounds {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, tt.want)
			}
			if got.Weight != tt.want.Weight {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.Tags, tt.want.Tags) {
				t.Errorf("ToAlgorithmInput() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	type args struct {
		r string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "valid config",
			args: args{
				r: "testdata/default_config.yml",
			},
			want: &Config{
				Name: "test",
				Inputs: []*Input{
					{
						Field:        "test",
						Distribution: "linear",
						Weight:       1,
					},
				},
			},
		},
		{
			name: "invalid yml",
			args: args{
				r: "testdata/invalid_config.yml",
			},
			want:    &Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// read file

			f, err := os.Open(tt.args.r)
			if err != nil {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := LoadConfig(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !cmp.Equal(got.Inputs, tt.want.Inputs) || got.Name != tt.want.Name {
				t.Log(cmp.Diff(got.Inputs, tt.want.Inputs))
				t.Errorf("LoadConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildCondition(t *testing.T) {
	type args struct {
		c *Condition
	}
	tests := []struct {
		name    string
		args    args
		want    algorithm.Condition
		wantErr bool
	}{
		{
			name: "invalid condition",
			args: args{
				c: &Condition{},
			},
			want:    nil,
			wantErr: true,
		},

		// Can't test the c.Not condition because algorithm.Condition is a func and can't be compared
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildCondition(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCondition() got = %v, want %v", got, tt.want)
			}
		})
	}
}
