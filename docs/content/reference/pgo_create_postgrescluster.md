---
title: pgo create postgrescluster
---
## pgo create postgrescluster

Create PostgresCluster with a given name

### Synopsis

Create basic PostgresCluster with a given name.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [create]

### Usage

```
pgo create postgrescluster CLUSTER_NAME [flags]
```

### Examples

```
# Create a postgrescluster with Postgres 15
pgo create postgrescluster hippo --pg-major-version 15

# Create a postgrescluster with backups disabled (only available in CPK v5.7+)
pgo create postgrescluster hippo --disable-backups --environment development

```
### Example output
```    
postgresclusters/hippo created
```

### Options

```
      --disable-backups        Disable backups
      --environment string     Set the Postgres cluster environment. Options: ['development', 'production'] (default "production")
  -h, --help                   help for postgrescluster
      --pg-major-version int   Set the Postgres major version
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

* [pgo create](/reference/pgo_create/)	 - Create a resource

