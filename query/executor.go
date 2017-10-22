package query

import (
	"fmt"

	"github.com/saracen/kubeql/query/ast"
	"github.com/saracen/kubeql/query/joiner"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/dynamic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Results struct {
	Headers []string
	Rows    []*Row
}

type Row struct {
	Columns []interface{}
}

func ExecuteQuery(c *rest.Config, query string) (*Results, error) {
	parser := NewStringParser(query)

	s, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	return executeSelectStatement(c, s)
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

	result := map[string]interface{} {
		i.name: i.data.Items[idx].UnstructuredContent(),
	}

	return joiner.Tuple(result)
}

func executeSelectStatement(c *rest.Config, s *ast.SelectStatement) (*Results, error) {
	pool := dynamic.NewDynamicClientPool(c)

	var iterators []joiner.Iterator
	for _, resource := range s.FromClause.Resources {
		gvk := schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}

		client, err := pool.ClientForGroupVersionKind(gvk)
		if err != nil {
			return nil, err
		}

		options := metav1.ListOptions{}
		list, err := client.Resource(&metav1.APIResource{Name: gvk.Kind, Group: gvk.Group, Version: gvk.Version, Namespaced: true}, s.FromClause.Namespace).List(options)
		if err != nil {
			return nil, err
		}

		data, ok := list.(*unstructured.UnstructuredList)
		if !ok {
			return nil, fmt.Errorf("Invalid kubernetes resource")
		}

		iterators = append(iterators, &UnstructuredListIterator{name: resource.Alias, data: data})
	}

	// set headers
	results := &Results{}
	for _, expr := range s.SelectClause.Expressions {
		alias := expr.Alias
		if alias == "" {
			alias = "?column?"
		}
		results.Headers = append(results.Headers, alias)
	}

	innerJoin := joiner.NewInnerJoin(iterators)
	for {
		if !innerJoin.HasNext() {
			break
		}

		item := innerJoin.Next()

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

			row.Columns = append(row.Columns, evaled)
		}
	}

	return results, nil
}