package algorithm

type Algorithm interface {
	Score(record map[string]float64) float64
}

type Factory func(inputs []*Input) (Algorithm, error)
