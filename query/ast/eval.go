package ast

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/saracen/kubeql/query/lexer"

	"github.com/reflect/filq"
	"k8s.io/client-go/util/jsonpath"
)

func EvalIsEmpty(expr Expr, data map[string]interface{}) (bool, error) {
	evaled, err := expr.Eval(data)
	if err != nil {
		return true, err
	}

	return checkIfEmpty(evaled), nil
}

func checkIfEmpty(data interface{}) (empty bool) {
	defer func() {
		// If reflect.Zero can't figure out a default, we're not empty
		if recover() != nil {
			empty = false
		}
	}()

	if data == nil {
		return true
	}

	val := reflect.ValueOf(data)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return val.Len() == 0
	}

	return data == reflect.Zero(val.Type()).Interface()
}

func (expr *ParenExpr) Eval(data map[string]interface{}) (interface{}, error) {
	evaled, err := expr.Expr.Eval(data)
	if err != nil {
		return nil, err
	}

	if expr.PathExpr != nil {
		return matchPathExpression(evaled, expr.PathExpr.Fields)
	}
	return evaled, nil
}

func (expr *Integer) Eval(data map[string]interface{}) (interface{}, error) {
	return expr.Val, nil
}

func (expr *Float) Eval(data map[string]interface{}) (interface{}, error) {
	return expr.Val, nil
}

func (expr *String) Eval(data map[string]interface{}) (interface{}, error) {
	return expr.Val, nil
}

func (expr *Boolean) Eval(data map[string]interface{}) (interface{}, error) {
	return expr.Val, nil
}

func (expr *Reference) Eval(data map[string]interface{}) (interface{}, error) {
	path := []string{expr.Name}
	if expr.PathExpr != nil {
		path = append(path, expr.PathExpr.Fields...)
	}

	return matchPathExpression(data, path)
}

func (expr *BinaryExpr) Eval(data map[string]interface{}) (val interface{}, err error) {
	lhs, err := expr.LHS.Eval(data)
	if err != nil {
		return nil, err
	}

	rhs, err := expr.RHS.Eval(data)
	if err != nil {
		return nil, err
	}

	return op(lhs, expr.Op, rhs), nil
}

func (expr *JsonPath) Eval(data map[string]interface{}) (interface{}, error) {
	evaled, err := expr.Expr.Eval(data)
	if err != nil {
		return nil, err
	}

	jp := jsonpath.New(expr.Path).AllowMissingKeys(true)
	if err = jp.Parse(expr.Path); err != nil {
		return nil, fmt.Errorf("jsonpath error, %v", err)
	}

	fullresults, err := jp.FindResults(evaled)
	if err != nil {
		return nil, fmt.Errorf("jsonpath error, %v", err)
	}

	ret := make([]interface{}, 0)
	for _, results := range fullresults {
		for _, result := range results {
			ret = append(ret, result.Interface())
		}
	}

	if expr.PathExpr != nil {
		return matchPathExpression(ret, expr.PathExpr.Fields)
	}

	return ret, nil
}

func (expr *JQ) Eval(data map[string]interface{}) (interface{}, error) {
	evaled, err := expr.Expr.Eval(data)
	if err != nil {
		return nil, err
	}

	outs, err := filq.Run(filq.NewContext(), expr.Path, evaled)
	if err != nil {
		return nil, err
	}

	if expr.PathExpr != nil {
		return matchPathExpression(outs, expr.PathExpr.Fields)
	}

	return outs, nil
}

func op(lhs interface{}, op Operator, rhs interface{}) (val interface{}) {
	defer func() {
		if r := recover(); r != nil {
			val = nil
		}
	}()

	// convert rhs to the same type as lhs
	switch rhs.(type) {
	// in the instance of floats, convert lhs to rhs type instead, so that
	// 1>=1.1 isn't true due to truncating
	case float64:
		lhs = reflect.ValueOf(lhs).Convert(reflect.TypeOf(rhs)).Interface()
	default:
		rhs = reflect.ValueOf(rhs).Convert(reflect.TypeOf(lhs)).Interface()
	}

	switch lexer.TokenType(op) {
	case lexer.Or:
		switch rhs := rhs.(type) {
		case bool:
			return lhs.(bool) || rhs
		}
	case lexer.And:
		switch rhs := rhs.(type) {
		case bool:
			return lhs.(bool) && rhs
		}
	case lexer.LessThan:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) < rhs
		case int64:
			return lhs.(int64) < rhs
		case float64:
			return lhs.(float64) < rhs
		}
	case lexer.LessThanEqual:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) <= rhs
		case int64:
			return lhs.(int64) <= rhs
		case float64:
			return lhs.(float64) <= rhs
		}
	case lexer.GreaterThan:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) > rhs
		case int64:
			return lhs.(int64) > rhs
		case float64:
			return lhs.(float64) > rhs
		}
	case lexer.GreaterThanEqual:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) >= rhs
		case int64:
			return lhs.(int64) >= rhs
		case float64:
			return lhs.(float64) >= rhs
		}
	case lexer.Equal:
		switch rhs := rhs.(type) {
		case bool:
			return lhs.(bool) == rhs
		case int:
			return lhs.(int) == rhs
		case int64:
			return lhs.(int64) == rhs
		case float64:
			return lhs.(float64) == rhs
		case string:
			return lhs.(string) == rhs
		}
	case lexer.NotEqual:
		switch rhs := rhs.(type) {
		case bool:
			return lhs.(bool) != rhs
		case int:
			return lhs.(int) != rhs
		case int64:
			return lhs.(int64) != rhs
		case float64:
			return lhs.(float64) != rhs
		case string:
			return lhs.(string) != rhs
		}
	case lexer.Subtract:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) - rhs
		case int64:
			return lhs.(int64) - rhs
		case float64:
			return lhs.(float64) - rhs
		}
	case lexer.Add:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) + rhs
		case int64:
			return lhs.(int64) + rhs
		case float64:
			return lhs.(float64) + rhs
		}
	case lexer.Multiply:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) * rhs
		case int64:
			return lhs.(int64) * rhs
		case float64:
			return lhs.(float64) * rhs
		}
	case lexer.Divide:
		switch rhs := rhs.(type) {
		case int:
			return lhs.(int) / rhs
		case int64:
			return lhs.(int64) / rhs
		case float64:
			return lhs.(float64) / rhs
		}
	}

	return nil
}

func matchPathExpression(content interface{}, fields []string) (interface{}, error) {
	if len(fields) == 0 {
		return content, nil
	}

	switch v := content.(type) {
	case map[string]interface{}:
		child, ok := v[fields[0]]
		if !ok {
			return nil, nil
		}

		return matchPathExpression(child, fields[1:])

	case []interface{}:
		// parse number
		i, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, nil
		}

		if i >= 0 && i < len(v) {
			return matchPathExpression(v[i], fields[1:])
		}
		return nil, nil
	}

	return content, nil
}
