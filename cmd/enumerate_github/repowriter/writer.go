package repowriter

// Writer is a simple interface for writing a repo. This interface is to
// abstract output formats for lists of repository urls.
type Writer interface {
	// Write outputs a single repository url.
	Write(repo string) error
}
