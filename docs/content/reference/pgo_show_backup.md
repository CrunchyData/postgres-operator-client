---
title: pgo show backup
---
## pgo show backup

Show backup information for a PostgresCluster

### Synopsis

Show backup information for a PostgresCluster from 'pgbackrest info' command.
	
### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]

### Usage

```
pgo show backup CLUSTER_NAME [flags]
```

### Examples

```
# Show every repository of the 'hippo' postgrescluster
pgo show backup hippo

# Show every repository of the 'hippo' postgrescluster as JSON
pgo show backup hippo --output=json

# Show one repository of the 'hippo' postgrescluster
pgo show backup hippo --repoName=repo1

```
### Example output
```
stanza: db
    status: ok
    cipher: none

    db (current)
        wal archive min/max (14): 000000010000000000000001/000000010000000000000004

        full backup: 20231023-201416F
            timestamp start/stop: 2023-10-23 20:14:16+00 / 2023-10-23 20:14:32+00
            wal start/stop: 000000010000000000000002 / 000000010000000000000002
            database size: 33.5MB, database backup size: 33.5MB
            repo1: backup set size: 4.2MB, backup size: 4.2MB
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

* [pgo show](/reference/pgo_show/)	 - Show PostgresCluster details

