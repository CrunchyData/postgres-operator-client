---
title: pgo show
---
## pgo show

Show PostgresCluster details

### Synopsis

Show allows you to display particular details related to the PostgresCluster.

### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]

### Usage

```
pgo show [flags]
```

### Examples

```
# Show the backup and HA output of the 'hippo' postgrescluster
pgo show hippo

```
### Example output
```
BACKUP

stanza: db
    status: ok
    cipher: none

    db (current)
        wal archive min/max (14): 000000010000000000000001/000000010000000000000003

        full backup: 20231030-183841F
            timestamp start/stop: 2023-10-30 18:38:41+00 / 2023-10-30 18:38:46+00
            wal start/stop: 000000010000000000000002 / 000000010000000000000002
            database size: 25.3MB, database backup size: 25.3MB
            repo1: backup set size: 3.2MB, backup size: 3.2MB

HA

+ Cluster: hippo-ha (7295822780081832000) -----+--------+---------+----+-----------+
| Member          | Host                       | Role   | State   | TL | Lag in MB |
+-----------------+----------------------------+--------+---------+----+-----------+
| hippo-00-cwqq-0 | hippo-00-cwqq-0.hippo-pods | Leader | running |  1 |           |
+-----------------+----------------------------+--------+---------+----+-----------+

```

### Options

```
  -h, --help   help for show
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
* [pgo show backup](/reference/pgo_show_backup/)	 - Show backup information for a PostgresCluster
* [pgo show ha](/reference/pgo_show_ha/)	 - Show 'patronictl list' for a PostgresCluster.
* [pgo show user](/reference/pgo_show_user/)	 - Show details for a PostgresCluster user.

