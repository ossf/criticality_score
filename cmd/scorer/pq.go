package main

import "container/heap"

// A RowItem is something to manage in a priority queue.
type RowItem struct {
	row   []string
	score float64
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds RowItems.
//
// The implementation of PriorityQueue is largely copied from the
// "container/heap" documentation.
type PriorityQueue []*RowItem

// Len implements the heap.Interface interface.
func (pq PriorityQueue) Len() int { return len(pq) }

// Less implements the heap.Interface interface.
func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].score > pq[j].score
}

// Swap implements the heap.Interface interface.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push implements the heap.Interface interface.
func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*RowItem)
	item.index = n
	*pq = append(*pq, item)
}

// Pop implements the heap.Interface interface.
func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// PushRow will add the given row into the priority queue with the score as the
// priority.
func (pq *PriorityQueue) PushRow(row []string, score float64) {
	heap.Push(pq, &RowItem{
		row:   row,
		score: score,
	})
}

// PopRow returns the row with the highest score.
func (pq *PriorityQueue) PopRow() []string {
	return heap.Pop(pq).(*RowItem).row
}
