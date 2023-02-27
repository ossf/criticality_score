package inputiter

import (
	"bufio"
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/ossf/criticality_score/internal/infile"
)

// Iterator returns an IterCloser for iterating across repositories for
// collecting signals.
//
// If only one arg is specified, the code will treat it as a file and attempt to
// open it. If the file doesn't exist, and is parseable as a URL the arg will be
// treated as a repo.
//
// If more than one arg is specified they are all considered to be repos.
//
// TODO: support the ability to force args to be interpreted as either a file,
// or a list of repos.
func New(args []string) (IterCloser[string], error) {
	if len(args) == 1 {
		// If there is 1 arg, attempt to open it as a file.
		fileOrRepo := args[0]
		urlParseFailed := false
		if _, err := url.Parse(fileOrRepo); err != nil {
			urlParseFailed = true
		}

		// Open the in-file for reading
		r, err := infile.Open(context.Background(), fileOrRepo)
		if err == nil {
			return &scannerIter{
				c:       r,
				scanner: bufio.NewScanner(r),
			}, nil
		}
		if urlParseFailed || !(errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrInvalid)) {
			// Only report errors if the file doesn't appear to be a URL, if the
			// filename doesn't exist, or the filename is invalid.
			return nil, err
		}
	}
	// If file loading failed, or there are 2 or more args, treat args as a list
	// of repos.
	return &sliceIter[string]{
		values: args,
	}, nil
}
