package ast

import (
	"github.com/saracen/kubeql/query/lexer"
)

type Visitor interface {
	Visit(Expr) Visitor
}

type ExprWalker interface {
	Walk(Visitor) Expr
}

type Expr interface {
	ExprWalker
	Eval(data map[string]interface{}) (interface{}, error)
}

type Operator lexer.TokenType

func (o Operator) Precedence() int {
	switch lexer.TokenType(o) {
	case lexer.Or:
		return 1
	case lexer.And:
		return 2
	case lexer.Equal, lexer.NotEqual, lexer.LessThan, lexer.LessThanEqual,
		lexer.GreaterThan, lexer.GreaterThanEqual:
		return 3
	case lexer.Add, lexer.Subtract:
		return 4
	case lexer.Multiply, lexer.Divide:
		return 5
	}
	return 0
}

func (o Operator) IsOperator() bool {
	switch lexer.TokenType(o) {
	case lexer.And, lexer.Or, lexer.Add, lexer.Subtract, lexer.Multiply,
		lexer.Divide, lexer.Equal, lexer.NotEqual, lexer.LessThan,
		lexer.LessThanEqual, lexer.GreaterThan, lexer.GreaterThanEqual:
		return true
	}

	return false
}

type BinaryExpr struct {
	Op  Operator
	LHS Expr
	RHS Expr
}

func (expr *BinaryExpr) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}
	expr.LHS.Walk(v)
	expr.RHS.Walk(v)

	return expr
}

type ParenExpr struct {
	Expr     Expr
	PathExpr *PathExpression
}

func (expr *ParenExpr) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	expr.Expr.Walk(v)

	return expr
}

type String struct {
	Val string
}

func (expr *String) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	return expr
}

type Integer struct {
	Val int
}

func (expr *Integer) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	return expr
}

type Float struct {
	Val float64
}

func (expr *Float) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	return expr
}

type Boolean struct {
	Val bool
}

func (expr *Boolean) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	return expr
}

type Reference struct {
	Name     string
	PathExpr *PathExpression
}

func (expr *Reference) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	return expr
}

type JsonPath struct {
	Expr     Expr
	Path     string
	PathExpr *PathExpression
}

func (expr *JsonPath) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	expr.Expr.Walk(v)

	return expr
}

type JQ struct {
	Expr     Expr
	Path     string
	PathExpr *PathExpression
}

func (expr *JQ) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	expr.Expr.Walk(v)

	return expr
}

func (expr *Subselect) Walk(v Visitor) Expr {
	if v = v.Visit(expr); v == nil {
		return expr
	}

	for _, expression := range expr.Select.SelectClause.Expressions {
		expression.Condition.Walk(v)
	}

	return expr
}

type inspector func(Expr) bool

func (f inspector) Visit(expr Expr) Visitor {
	if f(expr) {
		return f
	}
	return nil
}

func Inspect(walker ExprWalker, f func(Expr) bool) {
	walker.Walk(inspector(f))
}
