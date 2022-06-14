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
// If the registery does not have a Factory for the supplied name an error will
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
