package memsql

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/runner-mei/memsql/memcore"
	"github.com/xwb1989/sqlparser"
)

type Foreign struct {
	Drv string
	Conn *sql.DB
}

func (f *Foreign) From(ec *Context, tableName, tableAs string, where *sqlparser.Where) (io.Closer, memcore.Query, error) {
	sqlstr := "SELECT * FROM " + tableName
	if tableAs != "" {
		sqlstr = sqlstr + " AS " + tableAs
	}

	debuger := ec.Debuger.Table(tableName, tableAs, nil)
	if where != nil {
		sqlstr = sqlstr + " WHERE " + sqlparser.String(where)
		debuger.SetWhere(where.Expr)

		if f.Drv == "sqlite3" {
			sqlstr = strings.Replace(sqlstr, "true", "1", -1)
			sqlstr = strings.Replace(sqlstr, "false", "0", -1)
		}
	}

	rows, err := f.Conn.QueryContext(ec.Ctx, sqlstr)
	if err != nil {
		return nil, memcore.Query{}, err
	}

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, memcore.Query{}, err
	}

	// columnTypes, err := rows.ColumnTypes()
	// if err != nil {
	// 	return nil, memcore.Query{}, err
	// }

	var initFuncs = make([]func(*memcore.Value) interface{}, len(columnNames))
	var columns = make([]memcore.Column, len(columnNames))
	for idx := range columns {
		columns[idx].TableName = tableName
		columns[idx].TableAs = tableAs
		columns[idx].Name = columnNames[idx]
		initFuncs[idx] = func(value *memcore.Value) interface{} {
			return scanValue{
				value: value,
			}
		}
	}

	return rows, memcore.Query{
		Iterate: func() memcore.Iterator {
			var done = false
			var lastErr error

			return func(memcore.Context) (item memcore.Record, err error) {
				if done {
					err = lastErr
					return
				}
				if !rows.Next() {
					err = rows.Close()
					done = true
					if err != nil {
						lastErr = err
					} else {
						lastErr = memcore.ErrNoRows
						err = memcore.ErrNoRows
					}
					return
				}
				destValues := make([]memcore.Value, len(columns))
				dest := make([]interface{}, len(columns))
				for idx := range columns {
					dest[idx] = initFuncs[idx](&destValues[idx])
				}
				err = rows.Scan(dest...)
				if err != nil {
					rows.Close()
					done = true
					lastErr = err
					return
				}

				item.Columns = columns
				item.Values = destValues
				return
			}
		},
	}, nil
}

type scanValue struct {
	value *memcore.Value
}

func (sv scanValue) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case int8:
		sv.value.SetInt64(int64(v))
	case int16:
		sv.value.SetInt64(int64(v))
	case int32:
		sv.value.SetInt64(int64(v))
	case int64:
		sv.value.SetInt64(v)
	case int:
		sv.value.SetInt64(int64(v))
	case uint8:
		sv.value.SetUint64(uint64(v))
	case uint16:
		sv.value.SetUint64(uint64(v))
	case uint32:
		sv.value.SetUint64(uint64(v))
	case uint64:
		sv.value.SetUint64(v)
	case uint:
		sv.value.SetUint64(uint64(v))
	case float32:
		sv.value.SetFloat64(float64(v))
	case float64:
		sv.value.SetFloat64(v)
	case string:
		sv.value.SetString(v)
	case bool:
		sv.value.SetBool(v)
	case []byte:
		sv.value.SetString(string(v))
	default:
		return fmt.Errorf("unsupported type %T", v)
	}
	return nil
}