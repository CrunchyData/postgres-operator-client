---
title: pgo stop
---
## pgo stop

Stop cluster

### Synopsis

Stop sets the spec.shutdown field to true, allowing you to stop a PostgreSQL cluster.
The --force-conflicts flag may be required if the spec.shutdown field has been used before.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]

### Usage

```
pgo stop CLUSTER_NAME [flags]
```

### Examples

```
# Stop a 'hippo' postgrescluster.
pgo stop hippo

# Resolve ownership conflict
pgo stop hippo --force-conflicts

```
### Example output
```
postgresclusters/hippo stop initiated
```

### Options

```
      --force-conflicts   take ownership and overwrite the shutdown setting
  -h, --help              help for stop
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

* [pgo](/reference/)	 - pgo is a kubectl plugin for PGO, the open source Postgres Operator

