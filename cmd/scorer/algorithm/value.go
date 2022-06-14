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

// ConditionalValue wraps an Inner value, but will only return it if Exists
// is present. If Not is true, the value will be only be returned if Exists
// is absent.
type ConditionalValue struct {
	Not    bool
	Exists Field
	Inner  Value
}

// Value implements the Value interface.
func (cv *ConditionalValue) Value(fields map[string]float64) (float64, bool) {
	v, ok := cv.Inner.Value(fields)
	if !ok {
		return 0, false
	}
	_, exists := fields[cv.Exists.String()]
	if exists != cv.Not { // not XOR exists
		return v, true
	} else {
		return 0, false
	}
}
