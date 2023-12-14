package iterator

import (
	"bufio"
	"fmt"
	"io"
)

// scannerIter implements Iter[string] using a bufio.Scanner to iterate through
// lines in a file.
type scannerIter struct {
	c       io.Closer
	scanner *bufio.Scanner
}

func (i *scannerIter) Item() string {
	return i.scanner.Text()
}

func (i *scannerIter) Next() bool {
	return i.scanner.Scan()
}

func (i *scannerIter) Err() error {
	if err := i.scanner.Err(); err != nil {
		return fmt.Errorf("scanner: %w", i.scanner.Err())
	}
	return nil
}

func (i *scannerIter) Close() error {
	if err := i.c.Close(); err != nil {
		return fmt.Errorf("closer: %w", i.c.Close())
	}
	return nil
}

func Lines(r io.ReadCloser) IterCloser[string] {
	return &scannerIter{
		c:       r,
		scanner: bufio.NewScanner(r),
	}
}
