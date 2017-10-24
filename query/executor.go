package query

import (
	"fmt"

	"github.com/saracen/kubeql/query/ast"
	"github.com/saracen/kubeql/query/joiner"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Results struct {
	Headers []string
	Rows    []*Row
}

type Row struct {
	Columns []interface{}
}

type Session struct {
	pool      dynamic.ClientPool
	resources map[schema.GroupVersionKind]*unstructured.UnstructuredList
}

func ExecuteQuery(c *rest.Config, query string) (*Results, error) {
	parser := NewStringParser(query)

	s, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	session := &Session{
		dynamic.NewDynamicClientPool(c),
		make(map[schema.GroupVersionKind]*unstructured.UnstructuredList),
	}

	return executeSelectStatement(session, s, nil)
}

type UnstructuredListIterator struct {
	name string
	idx  int
	data *unstructured.UnstructuredList
}

func (i *UnstructuredListIterator) HasNext() bool {
	if i.idx < len(i.data.Items) {
		return true
	}

	return false
}

func (i *UnstructuredListIterator) Next() joiner.Tuple {
	if i.idx == len(i.data.Items) {
		i.idx = 0
	}

	idx := i.idx
	i.idx++

	result := map[string]interface{}{
		i.name: i.data.Items[idx].UnstructuredContent(),
	}

	return joiner.Tuple(result)
}

type ResultIterator struct {
	name string
	idx  int
	data *Results
}

func (i *ResultIterator) HasNext() bool {
	if i.idx < len(i.data.Rows) {
		return true
	}

	return false
}

func (i *ResultIterator) Next() joiner.Tuple {
	if i.idx == len(i.data.Rows) {
		i.idx = 0
	}

	idx := i.idx
	i.idx++

	kv := make(joiner.Tuple)

	row := i.data.Rows[idx]
	for idx, header := range i.data.Headers {
		kv[header] = row.Columns[idx]
	}

	result := make(joiner.Tuple)
	result[i.name] = kv

	return result
}

func getResourceIterators(session *Session, resources []*ast.FromResource) ([]joiner.Iterator, error) {
	var iterators []joiner.Iterator
	for _, resource := range resources {
		gvk := schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}

		data, ok := session.resources[gvk]
		if !ok {
			client, err := session.pool.ClientForGroupVersionKind(gvk)
			if err != nil {
				return nil, err
			}

			options := metav1.ListOptions{}
			list, err := client.Resource(&metav1.APIResource{Name: gvk.Kind, Group: gvk.Group, Version: gvk.Version, Namespaced: true}, resource.Namespace).List(options)
			if err != nil {
				return nil, err
			}

			data, ok = list.(*unstructured.UnstructuredList)
			if !ok {
				return nil, fmt.Errorf("Invalid kubernetes resource")
			}

			session.resources[gvk] = data
		}

		iterators = append(iterators, &UnstructuredListIterator{name: resource.Alias, data: data})
	}

	return iterators, nil
}

func prepareSubselects(session *Session, walker ast.ExprWalker) {
	ast.Inspect(walker, func(expr ast.Expr) bool {
		subselect, ok := expr.(*ast.Subselect)
		if !ok {
			return true
		}

		subselect.SelectEval = func(data map[string]interface{}) (interface{}, error) {
			results, err := executeSelectStatement(session, subselect.Select, data)
			if err != nil {
				return nil, err
			}

			if len(results.Rows) == 0 {
				return nil, nil
			}

			if len(results.Rows) > 1 {
				return nil, fmt.Errorf("more than one row returned by a subquery used as an expression")
			}

			if len(results.Rows[0].Columns) > 1 {
				return nil, fmt.Errorf("subquery must return only one column")
			}

			return results, nil
		}

		return false
	})
}

func executeSelectStatement(session *Session, s *ast.SelectStatement, data map[string]interface{}) (*Results, error) {
	prepareSubselects(session, s.SelectClause)
	if s.WhereClause != nil {
		prepareSubselects(session, s.WhereClause)
	}

	iterators, err := getResourceIterators(session, s.FromClause.Resources)
	if err != nil {
		return nil, err
	}

	for _, subselect := range s.FromClause.Subselects {
		results, err := executeSelectStatement(session, subselect.Select, data)
		if err != nil {
			return nil, err
		}
		iterators = append(iterators, &ResultIterator{name: subselect.Alias, data: results})
	}

	results := &Results{}
	innerJoin := joiner.NewInnerJoin(iterators)
	for {
		if !innerJoin.HasNext() {
			break
		}

		item := make(joiner.Tuple).Merge(data, innerJoin.Next())

		// Filter
		if s.WhereClause != nil && s.WhereClause.Condition != nil {
			empty, err := ast.EvalIsEmpty(s.WhereClause.Condition, item)
			if err != nil {
				return nil, err
			}
			if empty {
				continue
			}
		}

		row := &Row{}
		results.Rows = append(results.Rows, row)

		// Extract
		for _, expr := range s.SelectClause.Expressions {
			evaled, err := expr.Condition.Eval(item)
			if err != nil {
				return nil, err
			}

			if subres, ok := evaled.(*Results); ok {
				if expr.Alias == "" {
					expr.Alias = subres.Headers[0]
				}
				evaled = subres.Rows[0].Columns[0]
			}

			row.Columns = append(row.Columns, evaled)
		}
	}

	// set headers
	for _, expr := range s.SelectClause.Expressions {
		alias := expr.Alias
		if alias == "" {
			alias = "?column?"
		}
		results.Headers = append(results.Headers, alias)
	}

	return results, nil
}
