package joiner

type Tuple map[string]interface{}

func (t Tuple) Merge(tuples ...Tuple) Tuple {
	for _, tuple := range tuples {
		for key, value := range tuple {
			t[key] = value
		}
	}

	return t
}

type Iterator interface {
	Next() Tuple
	HasNext() bool
}

type Joiner interface {
	Iterator
}
