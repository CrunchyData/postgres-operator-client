---
title: pgo version
---
## pgo version

PGO client and operator versions

### Synopsis

Version displays the versions of the PGO client and the Crunchy Postgres Operator

### RBAC Requirements
    Resources                                       Verbs
    ---------                                       -----
    customresourcedefinitions.apiextensions.k8s.io  [get]

    Note: This RBAC needs to be cluster-scoped.

### Usage

```
pgo version [flags]
```

### Examples

```
# Request the version of the client and the operator
pgo version

```
### Example output
```
Client Version: v0.5.2
Operator Version: v5.7.0
```

### Options

```
      --client   If true, shows client version only (no server required).
  -h, --help     help for version
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

