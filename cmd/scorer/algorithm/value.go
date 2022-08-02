package algorithm

type Value interface {
	// Value takes in a set of fields does some work and returns either the
	// result and true to indicate success, or 0 and false to indicate
	// the result could not be generated.
	Value(fields map[string]float64) (float64, bool)
}

// Field implements the Value interface, but simply returns the raw value of
// the named field.
type Field string

func (f Field) String() string {
	return string(f)
}

// Value implements the Value interface.
func (f Field) Value(fields map[string]float64) (float64, bool) {
	v, ok := fields[string(f)]
	return v, ok
}

type Condition func(fields map[string]float64) bool

func NotCondition(c Condition) Condition {
	return func(fields map[string]float64) bool {
		return !c(fields)
	}
}

func ExistsCondition(f Field) Condition {
	return func(fields map[string]float64) bool {
		_, exists := fields[f.String()]
		return exists
	}
}

// ConditionalValue wraps an Inner value that will only be returned if the
// Condition returns true.
type ConditionalValue struct {
	Condition Condition
	Inner     Value
}

// Value implements the Value interface.
func (cv *ConditionalValue) Value(fields map[string]float64) (float64, bool) {
	v, ok := cv.Inner.Value(fields)
	if !ok {
		return 0, false
	}
	if cv.Condition(fields) {
		return v, true
	} else {
		return 0, false
	}
}
