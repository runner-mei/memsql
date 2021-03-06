package memcore

// Take returns a specified number of contiguous elements from the start of a
// collection.
func (q Query) Take(count int) Query {
	return Query{
		Iterate: func() Iterator {
			next := q.Iterate()
			n := count

			return func(ctx Context) (item Record, err error) {
				if n <= 0 {
					err = ErrNoRows
					return
				}

				n--
				return next(ctx)
			}
		},
	}
}

// TakeWhile returns elements from a collection as long as a specified
// condition is true. The element's index is used in the logic of the predicate
// function. The first argument of predicate represents the zero-based index of
// the element within collection. The second argument represents the element to
// test.
func (q Query) TakeWhile(predicate func(int, Record) bool) Query {
	return Query{
		Iterate: func() Iterator {
			next := q.Iterate()
			done := false
			index := 0

			return func(ctx Context) (item Record, err error) {
				if done {
					err = ErrNoRows
					return
				}

				for {
					item, err = next(ctx)
					if err != nil {
						if IsNoRows(err) {
							done = true
						}
						return
					}

					if predicate(index, item) {
						index++
						return
					}
				}
			}
		},
	}
}
