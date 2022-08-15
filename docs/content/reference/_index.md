---
title: Command Reference
aliases:
- /reference/pgo
weight: 100
---
## pgo

pgo is a kubectl plugin for PGO, the open source Postgres Operator

### Synopsis

pgo is a kubectl plugin for PGO, the open source Postgres Operator from Crunchy Data.

	https://github.com/CrunchyData/postgres-operator

### Options

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
  -h, --help                           help for pgo
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

* [pgo backup](/reference/pgo_backup/)	 - Backup cluster
* [pgo create](/reference/pgo_create/)	 - Create a resource
* [pgo delete](/reference/pgo_delete/)	 - Delete a resource
* [pgo restore](/reference/pgo_restore/)	 - Restore cluster
* [pgo show](/reference/pgo_show/)	 - Show PostgresCluster details
* [pgo support](/reference/pgo_support/)	 - Crunchy Support commands for PGO
* [pgo version](/reference/pgo_version/)	 - PGO client and operator versions

