---
title: "kubectl-pgo show backup"
---
## kubectl-pgo show backup

Show backup information for a PostgresCluster

### Synopsis

Show backup information for a PostgresCluster from 'pgbackrest info' command.

```
kubectl-pgo show backup CLUSTER_NAME [flags]
```

### Examples

```
  kubectl pgo show backup hippo
  kubectl pgo show backup hippo --output=json
  kubectl pgo show backup hippo --repoName=repo1
	
```

### Options

```
  -h, --help              help for backup
  -o, --output string     output format. types supported: text,json (default "text")
      --repoName string   Set the repository name for the command. example: repo1
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

* [kubectl-pgo show](/reference/kubectl-pgo_show/)	 - Show PostgresCluster details

