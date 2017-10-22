# kubeql

Kubeql, pronounced "cubicle", is a SQL-like query language for Kubernetes
resources.

It *might* be handy for simple queries, but at the moment, it is very much a toy
project for me to learn about parsers, lexers and evaluators.

Things you can do:

### Simple selections

`->` is currently used over dot-notation, as dot notation is commonly used in
JSONPath and "jq" like expressions. In contrast, `->` access is simple, and only
supports direct->path->access. It supports map and array access (`array->0->item`).

```
$ ./kubeql -execute "select pods->metadata->labels as labels from pods"

labels
------
{"app":"redmine-test-2-mariadb","pod-template-hash":"384399387"}
{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}
{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}
{"k8s-app":"event-exporter","pod-template-hash":"1421584133","version":"v0.1.5"}
...
```

The same `->` path expressions can be used for filtering.

```
$ ./kubeql -execute "select pods->metadata->labels as labels from pods where pods->metadata->labels->app"

labels
------
{"app":"redmine-test-2-mariadb","pod-template-hash":"384399387"}
{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}
{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}
{"app":"helm","name":"tiller","pod-template-hash":"1936853538"}
```
```
$ ./kubeql -execute "select pods->metadata->labels as labels from pods where pods->metadata->labels->app = 'helm'"

labels
------
{"app":"helm","name":"tiller","pod-template-hash":"1936853538"}
```

```
$ ./kubeql -execute "select pods->spec->containers->0->name as names from pods"

names
-----
"redmine-test-2-mariadb"
"redmine-test-2-redmine"
"redmine-test-2-redmine"
"event-exporter"
...
```

### Resource access

Kubeql can access non-core v1 resources by a fully qualified name
(eg: `apps/v1beta1/deployments`), and core v1 resources by their short-name
(pods, endpoints, services, configmaps, secrets, persistentvolumeclaims, events
etc).


```
$ ./kubeql -execute "select deployments->metadata->name as deployment_name FROM apps/v1beta1/deployments"

deployment_name
---------------
"redmine-test-2-mariadb"
"redmine-test-2-redmine"
"event-exporter"
"heapster-v1.4.2"
...
```

### Indexes

Kubeql **does not yet** fetch efficiently from the backend. In the future, I
hope I can use the label/field selector to fetch fewer results than required so
that there's less to be processed by the client.

### Namespaces

Using the `NAMESPACE` keyword will only fetch resources from the specified namespace.

`select deployments FROM apps/v1beta1/deployments NAMESPACE default`

### Joins

Kubectl at the moment only supports SQL ANSI-89 JOIN functionality, by selecting
from multiple tables.

```
$ ./kubeql -execute "select deployments->metadata->name as deployment_name, pods->metadata->name as pod_name FROM apps/v1beta1/deployments, pods where pods->metadata->labels->app = deployments->metadata->labels->app"

deployment_name          pod_name
---------------          --------
"redmine-test-2-mariadb" "redmine-test-2-mariadb-384399387-dz3xq"
"redmine-test-2-redmine" "redmine-test-2-redmine-411540601-320ws"
"redmine-test-2-redmine" "redmine-test-2-redmine-411540601-938tq"
"tiller-deploy"          "tiller-deploy-1936853538-hvjnm"
```

### JSONPath

Kubeql supports kubernetes' implementation of JSONPath templating.

```
$ ./kubeql -execute "select jsonpath(pods->metadata, '{.labels}') as labels from pods"

labels
------
[{"app":"redmine-test-2-mariadb","pod-template-hash":"384399387"}]
[{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}]
[{"app":"redmine-test-2-redmine","pod-template-hash":"411540601"}]
[{"k8s-app":"event-exporter","pod-template-hash":"1421584133","version":"v0.1.5"}]
[{"controller-revision-hash":"1419153066","k8s-app":"fluentd-gcp","kubernetes.io/cluster-service":"true","pod-template-generation":"1","version":"v2.0"}]
...
```

### JQ

Kubeql supports JQ-style selecting/filtering.

```
$ ./kubeql -execute "select jq(deployments->metadata, '.labels.app') as deployment_name FROM apps/v1beta1/deployments"

deployment_name
---------------
["redmine-test-2-mariadb"]
["redmine-test-2-redmine"]
[null]
[null]
["helm"]
```

Because `jsonpath` and `jq` return arrays, you can always combine this with
Kubeql's `->` path expressions to return a single result:

```
$ ./kubeql -execute "select jq(deployments->metadata, '.labels.app')->0 as deployment_name FROM apps/v1beta1/deployments"

deployment_name
---------------
"redmine-test-2-mariadb"
"redmine-test-2-redmine"
null
null
"helm"
```