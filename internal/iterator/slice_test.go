package iterator_test

import (
	"slices"
	"testing"

	"github.com/ossf/criticality_score/v2/internal/iterator"
)

func TestSliceIter_Empty(t *testing.T) {
	i := iterator.Slice[int]([]int{})

	if got := i.Next(); got {
		t.Errorf("Next() = %v; want false", got)
	}
}

func TestSliceIter_SingleEntry(t *testing.T) {
	want := 42
	i := iterator.Slice[int]([]int{want})

	if got := i.Next(); !got {
		t.Errorf("Next() = %v; want true", got)
	}
	if got := i.Item(); got != want {
		t.Errorf("Item() = %v; want %v", got, want)
	}
	if got := i.Next(); got {
		t.Errorf("Next()#2 = %v; want false", got)
	}
}

func TestSliceIter_MultiEntry(t *testing.T) {
	want := []int{1, 2, 3, 42, 1337}
	i := iterator.Slice[int](want)

	var got []int
	for i.Next() {
		got = append(got, i.Item())
	}

	if !slices.Equal(got, want) {
		t.Errorf("Iterator returned %v, want %v", got, want)
	}
}
