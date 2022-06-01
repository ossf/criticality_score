package projectrepo

import (
	"context"
	"errors"
	"net/url"
)

// ErrorNotFound is returned when there is no factory that can be used for a
// given URL.
var ErrorNotFound = errors.New("factory not found for url")

var globalResolver = &Resolver{}

// Resolver is used to resolve a Repo url against a set of Factory instances
// registered with the resolver.
type Resolver struct {
	fs []Factory
}

func (r *Resolver) findFactory(u *url.URL) Factory {
	for _, f := range r.fs {
		if f.Match(u) {
			return f
		}
	}
	return nil
}

// Register adds the factory f to the set of factories that can be used for
// resolving a url to a Repo.
func (r *Resolver) Register(f Factory) {
	r.fs = append(r.fs, f)
}

// Resolve takes a url u and returns a corresponding instance of Repo if a
// matching factory has been registered.
//
// If a matching factory can not be found an ErrorNotFound will be returned.
//
// The factory may also return an error.
func (r *Resolver) Resolve(ctx context.Context, u *url.URL) (Repo, error) {
	f := r.findFactory(u)
	if f == nil {
		return nil, ErrorNotFound
	}
	return f.New(ctx, u)
}

// Register the factory f with the global resolver.
func Register(f Factory) {
	globalResolver.Register(f)
}

// Resolve the given url u with the global resolver.
func Resolve(ctx context.Context, u *url.URL) (Repo, error) {
	return globalResolver.Resolve(ctx, u)
}
