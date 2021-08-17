package parser

import (
	"fmt"
	"reflect"

	"github.com/xwb1989/sqlparser"
	"github.com/runner-mei/memsql/memcore"
	"github.com/runner-mei/memsql/filter"
	"github.com/runner-mei/errors"
)

func parse(sqlstr string) (*sqlparser.Select, error) {
	stmt, err := sqlparser.Parse(sqlstr)
	if err != nil {
		return nil, err
	}
	// Otherwise do something with stmt
	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return nil, errors.New("only support select statement")
	}
	return selectStmt, nil
}

type Datasource struct {
	Table string
	As    string
}

type simpleExecuteContext struct {
	s    memcore.Storage
	stmt *sqlparser.Select
	ds   Datasource
}

func ExecuteSelectStatement(ec *simpleExecuteContext, stmt sqlparser.SelectStatement) (memcore.Query, error) {
	switch expr := stmt.(type) {
	case *sqlparser.Select:
			return ExecuteSelect(ec, expr)
	case *sqlparser.Union:
			return ExecuteUnion(ec, expr)
	case *sqlparser.ParenSelect:
		return ExecuteSelectStatement(ec, expr.Select) 
	default:
		return memcore.Query{}, fmt.Errorf("invalid select %+v of type %T", stmt, stmt)
	}
}

func ExecuteUnion(ec *simpleExecuteContext, stmt *sqlparser.Union) (memcore.Query, error) {
	left, err := ExecuteSelectStatement(ec, stmt.Left)
	if err != nil {
		return memcore.Query{}, err
	}
	right, err := ExecuteSelectStatement(ec, stmt.Right)
	if err != nil {
		return memcore.Query{}, err
	}

	var query memcore.Query
	switch stmt.Type {
	case sqlparser.UnionStr:
		query = left.Union(right)
	case sqlparser.UnionAllStr:
		query = left.UnionAll(right)
	// case sqlparser.UnionDistinctStr:
	default:
		return memcore.Query{}, fmt.Errorf("invalid union type %s", stmt.Type)
	}
	
	if len(stmt.OrderBy) > 0 {
		query, err = ExecuteOrderBy(ec, query, stmt.OrderBy)
		if err != nil {
			return memcore.Query{}, err
		}
	}
	
	if stmt.Limit != nil {
		query, err = ExecuteLimit(ec, query, stmt.Limit)
		if err != nil {
			return memcore.Query{}, err
		}
	}
	// if 	len(stmt.OrderBy) > 0 {
	// 	for idx := range stmt.OrderBy {
	// 		// Order represents an ordering expression.
	// 		type Order struct {
	// 			Expr      Expr
	// 			Direction string
	// 		}

	// 		// Order.Direction
	// 		const (
	// 			AscScr  = "asc"
	// 			DescScr = "desc"
	// 		)
	// 	}
	// }

	// if stmt.Limit != nil {
	// 	Offset, 
	// 	Rowcount Expr
	// }

	return query, nil
}

func ExecuteSelect(ec *simpleExecuteContext, stmt *sqlparser.Select) (memcore.Query, error) {
	if len(stmt.From) != 1 {
		return memcore.Query{}, fmt.Errorf("currently only one expression in from supported, got %v", len(stmt.From))
	}

	if stmt.Hints != "" {
		return memcore.Query{}, errors.New("currently unsupport hints")
	}

	if stmt.Lock != "" {
		return memcore.Query{}, errors.New("currently unsupport lock")
	}
	if stmt.Distinct != "" {
		return memcore.Query{}, errors.New("currently unsupport distinct")
	}

	// Where       *Where

	query, err := ExecuteTableExpression(ec, stmt.From[0], stmt.Where)
	if err != nil {
		return memcore.Query{}, errors.Wrap(err, "couldn't parse from expression")
	}

	if stmt.GroupBy != nil {
		query, err = ExecuteGroupBy(ec, query, stmt.GroupBy)
		if err != nil {
			return memcore.Query{}, err
		}

		if stmt.Having != nil {
			query, err = ExecuteHaving(ec, query, stmt.Having)
			if err != nil {
				return memcore.Query{}, err
			}
		}
	} else {
		if stmt.Having != nil {
			return memcore.Query{}, errors.New("currently unsupport having")
		}
	}

	if stmt.OrderBy != nil {
		query, err = ExecuteOrderBy(ec, query, stmt.OrderBy)
		if err != nil {
			return memcore.Query{}, err
		}
	}
	
	if stmt.Limit != nil {
		query, err = ExecuteLimit(ec, query, stmt.Limit)
		if err != nil {
			return memcore.Query{}, err
		}
	}

	return query, nil
}

func ExecuteTableExpression(ec *simpleExecuteContext, expr sqlparser.TableExpr, where *sqlparser.Where) (memcore.Query, error) {
	switch expr := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		return ExecuteAliasedTableExpression(ec, expr, where)
	//case *sqlparser.JoinTableExpr:
	//	return ParseJoinTableExpression(ec, expr, where)
	// case *sqlparser.ParenTableExpr:
	// 	return ParseTableExpression(expr.Exprs[0])
	default:
		return memcore.Query{}, fmt.Errorf("invalid table expression %+v of type %v", expr, reflect.TypeOf(expr))
	}
}

/*
func ExecuteJoinTableExpression(ec *simpleExecuteContext, expr *sqlparser.JoinTableExpr, where *sqlparser.Where) error {
// 	type JoinTableExpr struct {
// 	LeftExpr  TableExpr
// 	Join      string
// 	RightExpr TableExpr
// 	Condition JoinCondition
// }

  err := ExecuteAliasedTableExpression(ec, expr.LeftExpr, where)
  if err != nil {
  	return err
  }

  err = ExecuteAliasedTableExpression(ec, expr.RightExpr, where)
  if err != nil {
  	return err
  }

  return nil
}
*/

func ExecuteAliasedTableExpression(ec *simpleExecuteContext, expr *sqlparser.AliasedTableExpr, where *sqlparser.Where) (memcore.Query, error) {
	if len(expr.Partitions) > 0 {
		return memcore.Query{}, fmt.Errorf("invalid partitions in the table expression %+v", expr.Expr)
	}
	if expr.Hints != nil {
		return memcore.Query{}, fmt.Errorf("invalid index hits in the table expression %+v", expr.Expr)
	}
	switch subExpr := expr.Expr.(type) {
	case sqlparser.TableName:
		ec.ds.Table = subExpr.Name.String()
		if expr.As.IsEmpty() {
			ec.ds.As = ec.ds.Table
		} else {
			ec.ds.As = expr.As.String()
		}

		if ec.stmt.Where.Expr == nil {
			return ExecuteTable(ec, ec.ds, nil)
		}
		return ExecuteTable(ec, ec.ds, ec.stmt.Where.Expr)
		// if expr.As.IsEmpty() {
		// 	return nil, fmt.Errorf("table \"%v\" must have unique alias", subExpr.Name)
		// }
		// return logical.NewDataSource(subExpr.Name.String(), expr.As.String()), nil
	case *sqlparser.Subquery:
		return ExecuteSelectStatement(ec, subExpr.Select)
	default:
		return memcore.Query{}, fmt.Errorf("invalid aliased table expression %+v of type %v", expr.Expr, reflect.TypeOf(expr.Expr))
	}
}

func ExecuteTable(ec *simpleExecuteContext, ds Datasource, expr sqlparser.Expr) (memcore.Query, error) {
	_, tableExpr, err := SplitByColumnName(expr, ByTag())
	if err != nil {
		return memcore.Query{}, errors.Wrap(err, "couldn't resolve where '"+sqlparser.String(expr)+"'")
	}

	var f = func(filter.Context) (bool, error) {
		return true, nil
	}
	if tableExpr != nil {
		f, err = ToFilter(nil, tableExpr)
		if err != nil {
			return memcore.Query{}, errors.Wrap(err, "couldn't convert where '"+sqlparser.String(expr)+"'")
		}
	}

	query, err := ec.s.From(ec, ds.Table, f)
	if err != nil {
		return memcore.Query{}, err
	}

	return ExecuteWhere(ec, query, expr)
}

func ExecuteWhere(ec *simpleExecuteContext, query memcore.Query, expr sqlparser.Expr) (memcore.Query, error) {
  if expr == nil {
		return query, nil
  }

  // TODO: XXX

  // type Where Expr
	return query, nil
}

func ExecuteGroupBy(ec *simpleExecuteContext, query memcore.Query, groupBy sqlparser.GroupBy) (memcore.Query, error) {
  // type GroupBy []Expr

  // TODO: XXX
	return query, nil
}


func ExecuteHaving(ec *simpleExecuteContext, query memcore.Query, having *sqlparser.Where) (memcore.Query, error) {
		if having == nil {
			return query, nil
		}
		return ExecuteWhere(ec, query, ec.stmt.Where.Expr)
}

func ExecuteOrderBy(ec *simpleExecuteContext, query memcore.Query, orderBy sqlparser.OrderBy) (memcore.Query, error) {
  // TODO: XXX

	// 	for idx := range stmt.OrderBy {
	// 		// Order represents an ordering expression.
	// 		type Order struct {
	// 			Expr      Expr
	// 			Direction string
	// 		}

	// 		// Order.Direction
	// 		const (
	// 			AscScr  = "asc"
	// 			DescScr = "desc"
	// 		)
	// 	}
	return query, nil
}

func ExecuteLimit(ec *simpleExecuteContext, query memcore.Query, limit *sqlparser.Limit) (memcore.Query, error) {
  // TODO: XXX

	// if stmt.Limit != nil {
	// 	Offset, 
	// 	Rowcount Expr
	// }
	return query, nil
}