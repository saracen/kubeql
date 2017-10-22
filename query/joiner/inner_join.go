package joiner

type InnerJoin struct {
	Joiner

	iterators []Iterator
	current   []Tuple
}

func NewInnerJoin(iters []Iterator) *InnerJoin {
	return &InnerJoin{iterators: iters}
}

func (j *InnerJoin) HasNext() bool {
	for _, iter := range j.iterators {
		has := iter.HasNext()

		// if first pass and an iterator has no next value, then we have no next
		if j.current == nil && !has {
			return false
		}

		if has {
			return true
		}
	}

	return false
}

func (j *InnerJoin) Next() Tuple {
	if j.current == nil {
		j.current = make([]Tuple, len(j.iterators))

		for idx := range j.iterators {
			j.current[idx] = j.iterators[idx].Next()
		}

		return make(Tuple).Merge(j.current...)
	}

	for idx := len(j.iterators) - 1; idx >= 0; idx-- {
		if j.iterators[idx].HasNext() {
			j.current[idx] = j.iterators[idx].Next()
			break
		}

		j.current[idx] = j.iterators[idx].Next()
	}

	return make(Tuple).Merge(j.current...)
}