package memcore

// Join correlates the elements of two collection based on matching keys.
//
// A join refers to the operation of correlating the elements of two sources of
// information based on a common key. Join brings the two information sources
// and the keys by which they are matched together in one method call. This
// differs from the use of SelectMany, which requires more than one method call
// to perform the same operation.
//
// Join preserves the order of the elements of outer collection, and for each of
// these elements, the order of the matching elements of inner.
func (q Query) Join(inner Query,
	outerKeySelector func(Record) Value,
	innerKeySelector func(Record) Value,
	resultSelector func(outer Record, inner Record) Record) Query {

	return Query{
		Iterate: func() Iterator {
			outernext := q.Iterate()
			innernext := inner.Iterate()

			innerLookup := make(map[Value][]Record)
			for {
				innerItem, err := innernext()
				if err != nil {
					if !IsNoRows(err) {
						return func() (Record, error) {
							return Record{}, err
						}
					}
					break
				}

				innerKey := innerKeySelector(innerItem)
				innerLookup[innerKey] = append(innerLookup[innerKey], innerItem)
			}

			var outerItem Record
			var innerGroup []Record
			innerLen, innerIndex := 0, 0

			return func() (item Record, err error) {
				if innerIndex >= innerLen {
					has := false
					for !has {
						outerItem, err = outernext()
						if err != nil {
							return
						}

						innerGroup, has = innerLookup[outerKeySelector(outerItem)]
						innerLen = len(innerGroup)
						innerIndex = 0
					}
				}

				item = resultSelector(outerItem, innerGroup[innerIndex])
				innerIndex++
				return item, nil
			}
		},
	}
}
