---
title: pgo show user
---
## pgo show user

Show details for a PostgresCluster user.

### Synopsis

Show details for a PostgresCluster user. Only shows
details for the default user for a PostgresCluster
or for users defined on the PostgresCluster spec.
Use the "--show-connection-info" flag to get the
connection info, including password.

#### RBAC Requirements
    Resources  Verbs
    ---------  -----
    secrets       [list]

### Usage

```
pgo show user USER_NAME --cluster CLUSTER_NAME [flags]
```

### Examples

```
# Show non-sensitive contents of users for "hippo" cluster
pgo show user --cluster hippo

# Show non-sensitive contents of user "rhino" for "hippo" cluster
pgo show user rhino --cluster hippo

# Show connection info for user "rhino" for "hippo" cluster,
# including sensitive password info
pgo show user rhino --cluster hippo --show-connection-info

```
### Example output
```
# Showing all the users of the "hippo" cluster
CLUSTER  USERNAME
hippo    hippo
hippo    rhino

# Showing the connection info for user "hippo" of cluster "hippo"
WARNING: This command will show sensitive password information.
Are you sure you want to continue? (yes/no): yes

Connection information for hippo for hippo cluster
Connection info string:
    dbname=hippo host=hippo-primary.postgres-operator.svc port=5432 user=hippo password=<password>
Connection URL:
    postgres://<password>@hippo-primary.postgres-operator.svc:5432/hippo
```

### Options

```
  -h, --help                   help for user
      --show-connection-info   show sensitive user fields
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

* [pgo show](/reference/pgo_show/)	 - Show PostgresCluster details

