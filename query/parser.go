package query

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/saracen/kubeql/query/ast"
	"github.com/saracen/kubeql/query/lexer"
)

type SelectStatement struct {
	Fields    []string
	Namespace string
	Resource  string
}

type Parser struct {
	s     *lexer.Scanner
	input string
}

func NewStringParser(input string) *Parser {
	in := bytes.NewBufferString(input)

	return &Parser{s: lexer.NewScanner(in), input: input}
}

func (p *Parser) Parse() (stmt *ast.SelectStatement, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	statement := p.s.Peek()

	switch statement {
	case lexer.Select:
		p.match(lexer.Select)
		return p.SelectStatement(), nil

	default:
		return nil, fmt.Errorf("Expected SELECT")
	}
}

func (p *Parser) error(str string, offset int) {
	panic(fmt.Sprintf("%v (offset: %v) (%v <)", str, offset, strconv.Quote(p.input[:offset])))
}

func (p *Parser) match(token lexer.TokenType) string {
	t, offset, text := p.s.Scan()
	if t != token {
		p.error("unexpected token", offset)
	}

	return text
}

func (p *Parser) SelectStatement() *ast.SelectStatement {
	selectStatement := &ast.SelectStatement{}

	selectStatement.SelectClause = p.SelectClause()
	selectStatement.FromClause = p.FromClause()

	if p.s.Peek() == lexer.Where {
		selectStatement.WhereClause = p.WhereClause()
	}

	p.match(lexer.EOF)

	return selectStatement
}

func (p *Parser) SelectClause() *ast.SelectClause {
	expressions := []*ast.SelectExpression{
		p.SelectExpression(),
	}

	for p.s.Peek() == lexer.Comma {
		p.match(lexer.Comma)
		expressions = append(expressions, p.SelectExpression())
	}

	return &ast.SelectClause{Expressions: expressions}
}

func (p *Parser) FromClause() *ast.FromClause {
	p.match(lexer.From)

	resources := []*ast.FromResource{
		p.FromResource(),
	}

	for p.s.Peek() == lexer.Comma {
		p.match(lexer.Comma)
		resources = append(resources, p.FromResource())
	}

	var namespace string
	if p.s.Peek() == lexer.Namespace {
		p.match(lexer.Namespace)
		namespace = p.match(lexer.Ident)
	}

	return &ast.FromClause{Namespace: namespace, Resources: resources}
}

func (p *Parser) FromResource() *ast.FromResource {
	resource := &ast.FromResource{Version: "v1"}
	resource.Kind = p.match(lexer.Ident)

	if p.s.Peek() == lexer.Divide {
		p.match(lexer.Divide)
		resource.Version = p.match(lexer.Ident)
		resource.Group = resource.Kind
		p.match(lexer.Divide)
		resource.Kind = p.match(lexer.Ident)
	}

	resource.Alias = p.AsAlias(resource.Kind, false)

	return resource
}

func (p *Parser) WhereClause() *ast.WhereClause {
	where := &ast.WhereClause{}

	p.match(lexer.Where)
	where.Condition = p.Expression(1)

	return where
}

func (p *Parser) SelectExpression() *ast.SelectExpression {
	selectExpr := &ast.SelectExpression{}

	selectExpr.Condition = p.Expression(1)
	selectExpr.Alias = p.AsAlias("", true)

	return selectExpr
}

func (p *Parser) PathExpression() *ast.PathExpression {
	var fields []string
	for p.s.Peek() == lexer.Arrow {
		p.match(lexer.Arrow)

		switch p.s.Peek() {
		case lexer.Ident:
			fields = append(fields, p.match(lexer.Ident))
		case lexer.String:
			fields = append(fields, p.match(lexer.String))
		case lexer.Integer:
			fields = append(fields, p.match(lexer.Integer))
		}
	}

	return &ast.PathExpression{
		Fields: fields,
	}
}

func (p *Parser) AsAlias(def string, allowString bool) string {
	if p.s.Peek() == lexer.As {
		p.match(lexer.As)
	}

	token := p.s.Peek()
	if token == lexer.Ident {
		return p.match(lexer.Ident)
	}
	if allowString && token == lexer.String {
		return p.match(lexer.String)
	}

	return def
}

func (p *Parser) Expression(precedence int) ast.Expr {
	lhs := p.UnaryExpression()

	for {
		op := ast.Operator(p.s.Peek())
		if op.IsOperator() && op.Precedence() >= precedence {
			p.match(lexer.TokenType(op))
		} else {
			break
		}

		rhs := p.Expression(op.Precedence())
		lhs = &ast.BinaryExpr{Op: op, LHS: lhs, RHS: rhs}
	}

	return lhs
}

func (p *Parser) UnaryExpression() ast.Expr {
	token := p.s.Peek()

	switch token {
	case lexer.OpenParenthesis:
		p.match(lexer.OpenParenthesis)
		expr := p.Expression(1)
		p.match(lexer.CloseParenthesis)

		paren := &ast.ParenExpr{Expr: expr}
		if p.s.Peek() == lexer.Arrow {
			paren.PathExpr = p.PathExpression()
		}

		return paren

	case lexer.String:
		return &ast.String{Val: p.match(lexer.String)}

	case lexer.Integer:
		num, _ := strconv.Atoi(p.match(lexer.Integer))
		return &ast.Integer{Val: num}

	case lexer.Float:
		num, _ := strconv.ParseFloat(p.match(lexer.Float), 64)
		return &ast.Float{Val: num}

	case lexer.True:
		p.match(lexer.True)
		return &ast.Boolean{Val: true}

	case lexer.False:
		p.match(lexer.False)
		return &ast.Boolean{Val: false}

	case lexer.Ident:
		name := p.match(lexer.Ident)
		ref := &ast.Reference{Name: name}
		if p.s.Peek() == lexer.Arrow {
			ref.PathExpr = p.PathExpression()
		}

		return ref

	case lexer.JsonPath:
		p.match(lexer.JsonPath)
		p.match(lexer.OpenParenthesis)
		expr := p.Expression(1)
		p.match(lexer.Comma)
		path := p.match(lexer.String)
		p.match(lexer.CloseParenthesis)

		jsonpath := &ast.JsonPath{Expr: expr, Path: path}
		if p.s.Peek() == lexer.Arrow {
			jsonpath.PathExpr = p.PathExpression()
		}

		return jsonpath

	case lexer.Jq:
		p.match(lexer.Jq)
		p.match(lexer.OpenParenthesis)
		expr := p.Expression(1)
		p.match(lexer.Comma)
		path := p.match(lexer.String)
		p.match(lexer.CloseParenthesis)

		jq := &ast.JQ{Expr: expr, Path: path}
		if p.s.Peek() == lexer.Arrow {
			jq.PathExpr = p.PathExpression()
		}

		return jq

	default:
		_, offset, _ := p.s.Scan()
		p.error("unexpected token in expression", offset)
	}

	return nil
}
