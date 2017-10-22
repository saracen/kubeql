package ast

type SelectStatement struct {
	SelectClause *SelectClause
	FromClause   *FromClause
	WhereClause  *WhereClause
}

type SelectClause struct {
	Expressions []*SelectExpression
}

type SelectExpression struct {
	Alias     string
	Condition Expr
}

type PathExpression struct {
	Fields []string
}

type FromClause struct {
	Namespace string
	Resources []*FromResource
}

type FromResource struct {
	Alias   string
	Group   string
	Version string
	Kind    string
}

type WhereClause struct {
	Condition Expr
}