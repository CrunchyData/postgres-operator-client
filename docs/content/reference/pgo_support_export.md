---
title: pgo support export
---
## pgo support export

Export a snapshot of a PostgresCluster

### Synopsis

The support export tool will collect information that is commonly necessary for troubleshooting a
PostgresCluster.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    configmaps                                          [list]
    cronjobs.batch                                      [list]
    deployments.apps                                    [list]
    endpoints                                           [list]
    events                                              [get list]
    ingresses.networking.k8s.io                         [list]
    jobs.batch                                          [list]
    limitranges                                         [list]
    namespaces                                          [get]
    networkpolicies.networking.k8s.io                   [list]
    nodes                                               [list]
    persistentvolumeclaims                              [list]
    poddisruptionbudgets.policy                         [list]
    pods                                                [list]
    pods/exec                                           [create]
    pods/log                                            [get]
    postgresclusters.postgres-operator.crunchydata.com  [get]
    replicasets.apps                                    [list]
    serviceaccounts                                     [list]
    services                                            [list]
    statefulsets.apps                                   [list]

    Note: This RBAC needs to be cluster-scoped to retrieve information on nodes and postgresclusters.

### Event Capture
    Support export captures all Events in the PostgresCluster's Namespace.
    Event duration is determined by the '--event-ttl' setting of the Kubernetes
    API server. Default is 1 hour.
    - https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/

### Usage

```
pgo support export CLUSTER_NAME [flags]
```

### Examples

```
# Short Flags
kubectl pgo support export daisy -o . -l 2

# Long Flags
kubectl pgo support export daisy --output . --pg-logs-count 2

# Monitoring namespace override
# This is only required when monitoring is not deployed in the PostgresCluster's namespace.
kubectl pgo support export daisy --monitoring-namespace another-namespace --output .

# Operator namespace override
# This is only required when the Operator is not deployed in the PostgresCluster's namespace.
# This is used for getting the logs and specs for the operator pod(s).
kubectl pgo support export daisy --operator-namespace another-namespace --output .

```
### Example output
```
┌────────────────────────────────────────────────────────────────
| PGO CLI Support Export Tool
| The support export tool will collect information that is
| commonly necessary for troubleshooting a PostgresCluster.
| Note: No data or k8s secrets are collected.
| However, kubectl is used to list plugins on the user's machine.
└────────────────────────────────────────────────────────────────
Collecting PGO CLI version...
Collecting names and namespaces for PostgresClusters...
Collecting current Kubernetes context...
Collecting Kubernetes version...
Collecting nodes...
Collecting namespace...
Collecting PostgresCluster...
Collecting statefulsets...
Collecting deployments...
Collecting replicasets...
Collecting jobs...
Collecting cronjobs...
Collecting poddisruptionbudgets...
Collecting pods...
Collecting persistentvolumeclaims...
Collecting configmaps...
Collecting services...
Collecting endpoints...
Collecting serviceaccounts...
Collecting ingresses...
Collecting networkpolicies...
Collecting limitranges...
Collecting events...
Collecting Postgres logs...
Collecting pgBackRest logs...
Collecting Patroni logs...
Collecting pgBackRest Repo Host logs...
Collecting PostgresCluster pod logs...
Collecting monitoring pod logs...
Collecting operator pod logs...
Collecting Patroni info...
Collecting pgBackRest info...
Collecting processes...
Collecting system times from containers...
Collecting list of kubectl plugins...
Collecting PGO CLI logs...
┌────────────────────────────────────────────────────────────────
| Archive file size: 0.02 MiB
| Email the support export archive to support@crunchydata.com
| or attach as a email reply to your existing Support Ticket
└────────────────────────────────────────────────────────────────
```

### Options

```
  -h, --help                          help for export
      --monitoring-namespace string   Monitoring namespace override
      --operator-namespace string     Operator namespace override
  -o, --output string                 Path to save export tarball
  -l, --pg-logs-count int             Number of pg_log files to save (default 2)
```

### Options inherited from parent commands

```
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "$HOME/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [pgo support](/reference/pgo_support/)	 - Crunchy Support commands for PGO

