package iterator

import "fmt"

type batchIter[T any] struct {
	input     IterCloser[T]
	lastErr   error
	item      []T
	batchSize int
}

func (i *batchIter[T]) nextBatch() ([]T, error) {
	var batch []T

	for i.input.Next() {
		item := i.input.Item()
		batch = append(batch, item)
		if len(batch) >= i.batchSize {
			break
		}
	}
	if err := i.input.Err(); err != nil {
		// The input iterator failed, so return an error.
		return nil, fmt.Errorf("input iter: %w", err)
	}
	if len(batch) == 0 {
		// We've passed the end.
		return nil, nil
	}
	return batch, nil
}

func (i *batchIter[T]) Item() []T {
	return i.item
}

func (i *batchIter[T]) Next() bool {
	if i.lastErr != nil {
		// Stop if we've encountered an error.
		return false
	}
	batch, err := i.nextBatch()
	if err != nil {
		i.lastErr = err
		return false
	}
	if len(batch) == 0 {
		// We are also done at this point.
		return false
	}
	i.item = batch
	return true
}

func (i *batchIter[T]) Err() error {
	return i.lastErr
}

func (i *batchIter[T]) Close() error {
	if err := i.input.Close(); err != nil {
		return fmt.Errorf("input close: %w", i.input.Close())
	}
	return nil
}

func Batch[T any](input IterCloser[T], batchSize int) IterCloser[[]T] {
	return &batchIter[T]{
		input:     input,
		batchSize: batchSize,
	}
}
