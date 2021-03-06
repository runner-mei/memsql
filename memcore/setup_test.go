package memcore

import (
	"fmt"
	"testing"

	"github.com/runner-mei/memsql/vm"
)

var MustToValue = vm.MustToValue

func mkCtx() Context {
	return nil
}

func makeRecord(value int64) Record {
	return Record{
		Columns: []Column{{Name: "c1"}},
		Values: []Value{
			vm.IntToValue(value),
		},
	}
}

func makeRecords(value ...int64) []Record {
	results := []Record{}
	for _, v := range value {
		results = append(results, makeRecord(v))
	}
	return results
}

func fromInts(input ...int64) Query {
	return FromRecords(makeRecords(input...))
}

func fromInt2(inputs [][2]int64) Query {
	results := []Record{}
	for _, value := range inputs {
		record := Record{
			Columns: []Column{{Name: "c1"}, {Name: "c2"}},
			Values: []Value{
				vm.IntToValue(value[0]),
				vm.IntToValue(value[1]),
			},
		}
		results = append(results, record)
	}
	return FromRecords(results)
}

func makeRecordWithStr(value string) Record {
	return Record{
		Columns: []Column{{Name: "c1"}},
		Values:  []Value{{Type: vm.ValueString, Str: value}},
	}
}

func makeRecordsWithStrings(value ...string) []Record {
	results := []Record{}
	for _, v := range value {
		results = append(results, Record{
			Columns: []Column{{Name: "c1"}},
			Values:  []Value{vm.StringToValue(v)},
		})
	}
	return results
}

func fromStrings(input ...string) Query {
	return FromRecords(makeRecordsWithStrings(input...))
}

type foo struct {
	f1 int
	f2 bool
	f3 string
}

func (f foo) Iterate() Iterator {
	i := 0

	return func(ctx Context) (item Record, err error) {
		switch i {
		case 0:
			item = Record{
				Columns: []Column{{Name: "c1"}},
				Values:  []Value{vm.IntToValue(int64(f.f1))},
			}
			err = nil
		case 1:
			item = Record{
				Columns: []Column{{Name: "c1"}},
				Values:  []Value{vm.BoolToValue(f.f2)},
			}
			err = nil
		case 2:
			item = Record{
				Columns: []Column{{Name: "c1"}},
				Values:  []Value{vm.StringToValue(f.f3)},
			}
			err = nil
		default:
			err = ErrNoRows
		}

		i++
		return
	}
}

// func (f foo) CompareTo(c Comparable) int {
// 	a, b := f.f1, c.(foo).f1

// 	if a < b {
// 		return -1
// 	} else if a > b {
// 		return 1
// 	}

// 	return 0
// }

func toSlice(q Query) (result []Record) {
	result, err := q.Results(mkCtx())
	if err != nil {
		panic(err)
	}
	return result
}

func validateQuery(q Query, output []Record) bool {
	next := q.Iterate()
	ctx := mkCtx()

	for _, oitem := range output {
		qitem, err := next(ctx)
		if err != nil {
			panic(err)
		}

		ok, err := oitem.EqualTo(qitem, vm.EmptyCompareOption())
		if err != nil {
			panic(err)
		}
		if !ok {
			return false
		}
	}

	_, err := next(ctx)
	if err != nil {
		if !IsNoRows(err) {
			panic(err)
		}
	} else {
		return false
	}

	_, err = next(ctx)
	if err != nil {
		if IsNoRows(err) {
			return true
		}
		panic(err)
	} else {
		return false
	}
}

func mustPanicWithError(t *testing.T, expectedErr string, f func()) {
	defer func() {
		r := recover()
		err := fmt.Sprintf("%s", r)
		if err != expectedErr {
			t.Fatalf("got=[%v] expected=[%v]", err, expectedErr)
		}
	}()
	f()
}
