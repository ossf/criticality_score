package projectrepo

import (
	"context"
	"net/url"
)

// Repo is the core interface representing a project's source repository.
type Repo interface {
	URL() *url.URL
}

// Factory is used to obtain new instances of Repo.
type Factory interface {
	// New returns a new instance of Repo for the supplied URL.
	//
	// If the project is not valid for use, or there is an issue creating the
	// the Project will be nil and an error will be returned.
	New(context.Context, *url.URL) (Repo, error)

	// Match returns true if this factory can create a new instance of Repo
	// repository for the given repository URL.
	Match(*url.URL) bool
}
