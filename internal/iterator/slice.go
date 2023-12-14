package iterator

// sliceIter implements iter using a slice for iterating.
type sliceIter[T any] struct {
	values []T
	next   int
}

func (i *sliceIter[T]) Item() T {
	return i.values[i.next-1]
}

func (i *sliceIter[T]) Next() bool {
	if i.next <= len(i.values) {
		i.next++
	}
	return i.next <= len(i.values)
}

func (i *sliceIter[T]) Err() error {
	return nil
}

func (i *sliceIter[T]) Close() error {
	return nil
}

func Slice[T any](slice []T) IterCloser[T] {
	return &sliceIter[T]{
		values: slice,
	}
}
