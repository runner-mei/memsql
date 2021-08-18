package memcore

import "sort"

type comparer func(Value, Value) int

type order struct {
	selector func(Record) Value
	compare  comparer
	desc     bool
}

// OrderedQuery is the type returned from OrderBy, OrderByDescending ThenBy and
// ThenByDescending functions.
type OrderedQuery struct {
	Query
	original Query
	orders   []order
}

// OrderBy sorts the elements of a collection in ascending order. Elements are
// sorted according to a key.
func (q Query) OrderBy(selector func(Record) Value) OrderedQuery {
	return OrderedQuery{
		orders:   []order{{selector: selector}},
		original: q,
		Query: Query{
			Iterate: func() Iterator {
				items, err := q.sort([]order{{selector: selector}})
				if err != nil {
					return func() (Record, error) {
						return Record{}, err
					}
				}
				len := len(items)
				index := 0

				return func() (item Record, err error) {
					if index < len {
						item = items[index]
						index++
						return
					}
					err = ErrNoRows
					return
				}
			},
		},
	}
}

// OrderByDescending sorts the elements of a collection in descending order.
// Elements are sorted according to a key.
func (q Query) OrderByDescending(selector func(Record) Value) OrderedQuery {
	return OrderedQuery{
		orders:   []order{{selector: selector, desc: true}},
		original: q,
		Query: Query{
			Iterate: func() Iterator {
				items, err := q.sort([]order{{selector: selector, desc: true}})
				if err != nil {
					return func() (Record, error) {
						return Record{}, err
					}
				}
				length := len(items)
				index := 0

				return func() (item Record, err error) {
					if index < length {
						item = items[index]
						index++
						return
					}

					err = ErrNoRows
					return
				}
			},
		},
	}
}

// ThenBy performs a subsequent ordering of the elements in a collection in
// ascending order. This method enables you to specify multiple sort criteria by
// applying any number of ThenBy or ThenByDescending methods.
func (oq OrderedQuery) ThenBy(selector func(Record) Value) OrderedQuery {
	return OrderedQuery{
		orders:   append(oq.orders, order{selector: selector}),
		original: oq.original,
		Query: Query{
			Iterate: func() Iterator {
				items, err := oq.original.sort(append(oq.orders, order{selector: selector}))
				if err != nil {
					return func() (Record, error) {
						return Record{}, err
					}
				}
				length := len(items)
				index := 0

				return func() (item Record, err error) {
					if index < length {
						item = items[index]
						index++
						return
					}

					err = ErrNoRows
					return
				}
			},
		},
	}
}

// ThenByDescending performs a subsequent ordering of the elements in a
// collection in descending order. This method enables you to specify multiple
// sort criteria by applying any number of ThenBy or ThenByDescending methods.
func (oq OrderedQuery) ThenByDescending(selector func(Record) Value) OrderedQuery {
	return OrderedQuery{
		orders:   append(oq.orders, order{selector: selector, desc: true}),
		original: oq.original,
		Query: Query{
			Iterate: func() Iterator {
				items, err := oq.original.sort(append(oq.orders, order{selector: selector, desc: true}))
				if err != nil {
					return func() (Record, error) {
						return Record{}, err
					}
				}
				length := len(items)
				index := 0

				return func() (item Record, err error) {
					if index < length {
						item = items[index]
						index++
						return
					}

					err = ErrNoRows
					return
				}
			},
		},
	}
}

// Sort returns a new query by sorting elements with provided less function in
// ascending order. The comparer function should return true if the parameter i
// is less than j. While this method is uglier than chaining OrderBy,
// OrderByDescending, ThenBy and ThenByDescending methods, it's performance is
// much better.
func (q Query) Sort(less func(i, j Record) bool) Query {
	return Query{
		Iterate: func() Iterator {
			items, err := q.lessSort(less)
			if err != nil {
				return func() (Record, error) {
					return Record{}, err
				}
			}
			length := len(items)
			index := 0

			return func() (item Record, err error) {
				if index < length {
					item = items[index]
					index++
					return
				}

				err = ErrNoRows
				return
			}
		},
	}
}

type sorter struct {
	items []Record
	less  func(i, j Record) bool
}

func (s sorter) Len() int {
	return len(s.items)
}

func (s sorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

func (s sorter) Less(i, j int) bool {
	return s.less(s.items[i], s.items[j])
}

func (q Query) sort(orders []order) (r []Record, err error) {
	next := q.Iterate()
	for {
		item, err := next()
		if err != nil {
			if IsNoRows(err) {
				break
			}
			return nil, err
		}

		r = append(r, item)
	}

	if len(r) == 0 {
		return
	}

	for i := range orders {
		if orders[i].compare != nil {
			continue
		}
		orders[i].compare = func(a Value, b Value) int {
			ret, err := a.CompareTo(b, emptyCompareOption)
			if err != nil {
				panic(err)
			}
			return ret
		}
	}

	s := sorter{
		items: r,
		less: func(i, j Record) bool {
			for _, order := range orders {
				x, y := order.selector(i), order.selector(j)
				switch order.compare(x, y) {
				case 0:
					continue
				case -1:
					return !order.desc
				default:
					return order.desc
				}
			}
			return false
		}}

	sort.Sort(s)
	return
}

func (q Query) lessSort(less func(i, j Record) bool) (r []Record, err error) {
	next := q.Iterate()
	for {
		item, err := next()
		if err != nil {
			if IsNoRows(err) {
				break
			}
			return nil, err
		}

		r = append(r, item)
	}

	s := sorter{items: r, less: less}

	sort.Sort(s)
	return
}
