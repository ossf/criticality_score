package algorithm

import "testing"

func TestBoundsApply(t *testing.T) {
	tests := []struct {
		name string
		b    Bounds
		v    float64
		want float64
	}{
		{
			"regular test",
			Bounds{
				0,
				10,
				false,
			},
			7,
			7,
		},
		{
			"SmallerIsBetter equals true",
			Bounds{
				0,
				10,
				true,
			},
			7,
			3,
		},
		{
			"lower is not 0",
			Bounds{
				40,
				80,
				false,
			},
			50,
			10,
		},
		{
			"upper bound > lower bound", // should this test work?
			Bounds{
				40,
				20,
				false,
			},
			30,
			0,
		},
		{
			"upper bound == lower bound", // similar question as above, should this work?
			Bounds{
				40,
				40,
				false,
			},
			40,
			0,
		},
		{
			"upper bound == lower bound and SmallerIsBetter is true", // same question as above
			Bounds{
				40,
				40,
				true,
			},
			40,
			0,
		},
		{
			"v is negative", // can v be negative?
			Bounds{
				0,
				10,
				false,
			},
			-10,
			0,
		},
		{
			"v is lower than lower bound and SmallerIsBetter is true",
			Bounds{
				20,
				30,
				true,
			},
			15,
			10,
		},
		{
			"v is greater than upper bound and SmallerIsBetter is true",
			Bounds{
				0,
				10,
				true,
			},
			20,
			0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.b.Apply(test.v); got != test.want {
				t.Errorf("Apply(%v) = %v, want %v", test.v, got, test.want)
			}
		})
	}
}

func TestInputValue(t *testing.T) {
	type fields struct {
		Source       Value
		Bounds       *Bounds
		Distribution *Distribution
		Tags         []string
		Weight       float64
	}
	type want struct {
		val float64
		ok  bool
	}

	//nolint:govet
	tests := []struct {
		name   string
		input  fields
		fields map[string]float64
		want   want
	}{
		{
			name: "regular test",
			input: fields{
				Source:       Field("test"),
				Distribution: LookupDistribution("linear"),
			},
			fields: map[string]float64{"test": 10},

			want: want{10, true},
		},
		{
			name: "invalid Field",
			input: fields{
				Source: Field("test2"),
			},
			fields: map[string]float64{"test": 10},

			want: want{0, false},
		},
		{
			name: "bounds not equal to nil",
			input: fields{
				Source: Field("test"),
				Bounds: &Bounds{
					Lower:           0,
					Upper:           10,
					SmallerIsBetter: false,
				},
				Distribution: LookupDistribution("linear"),
			},
			fields: map[string]float64{"test": 5},

			want: want{0.5, true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			i := &Input{
				Source:       test.input.Source,
				Bounds:       test.input.Bounds,
				Distribution: test.input.Distribution,
				Tags:         test.input.Tags,
				Weight:       test.input.Weight,
			}
			wantVal, wantBool := i.Value(test.fields)
			if wantVal != test.want.val {
				t.Errorf("Value() wantVal = %v, want %v", wantVal, test.want.val)
			}
			if wantBool != test.want.ok {
				t.Errorf("Value() wantBool = %v, want %v", wantBool, test.want.ok)
			}
		})
	}
}
