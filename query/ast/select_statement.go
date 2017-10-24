package ast

type SelectStatement struct {
	SelectClause *SelectClause
	FromClause   *FromClause
	WhereClause  *WhereClause
}

type SelectClause struct {
	Expressions []*SelectExpression
}

func (stmt *SelectClause) Walk(v Visitor) Expr {
	for _, expression := range stmt.Expressions {
		expression.Condition.Walk(v)
	}
	return nil
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

func (stmt *WhereClause) Walk(v Visitor) Expr {
	stmt.Condition.Walk(v)

	return nil
}

type Subselect struct {
	Select *SelectStatement

	SelectEval func(map[string]interface{}) (interface{}, error)
}
