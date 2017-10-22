package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/saracen/kubeql/query"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	var execute = flag.String("execute", "", "query to execute")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	results, err := query.ExecuteQuery(config, *execute)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 8, 1, ' ', 0)

	fmt.Fprintln(writer, strings.Join(results.Headers, "\t"))
	var underscore []string
	for _, header := range results.Headers {
		underscore = append(underscore, strings.Repeat("-", len(header)))
	}
	fmt.Fprintln(writer, strings.Join(underscore, "\t"))

	for _, row := range results.Rows {
		c := make([]string, len(row.Columns))
		for i, column := range row.Columns {
			pretty, _ := json.Marshal(column)

			c[i] = string(pretty)
		}

		fmt.Fprintln(writer, strings.Join(c, "\t"))
	}

	writer.Flush()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
