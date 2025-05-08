<h1 align="center"><code>pgo</code>, the Postgres Operator CLI from Crunchy Data</h1>
<p align="center">
  <img width="150" src="./docs/static/logos/pgo.svg" alt="pgo: The CLI for the Postgres Operator from Crunchy Data"/>
</p>

Welcome to the repository for `pgo`, the Command Line Interface (CLI) for
the [Crunchy Postgres Operator](https://github.com/CrunchyData/postgres-operator)!
Built as a `kubectl` plugin, the `pgo` CLI facilitates the creation and management of PostgreSQL
clusters created using the Crunchy Postgres Operator. For more information about using the CLI and
the various commands available, please see the
[`pgo` CLI documentation](https://access.crunchydata.com/documentation/postgres-operator-client/latest).

## Install `pgo`

The following steps will allow you to download and install the `pgo`
[kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
in your local environment.

### Prerequisites

Depending on your deployment type, Kubernetes or OpenShift, `kubectl` or `oc`
must be installed and configured in your environment. For the purposes of these
instructions we will be using the `kubectl` client. The `pgo`
[kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
will use the role-based access controls (RBAC) that are configured for your
`kubectl` client.

### Download the binary

The `kubectl-pgo` binary is available either through the Crunchy Data
[Access Portal](https://access.crunchydata.com/downloads/browse/containers/postgres-operator/cli/) or via
[GitHub](https://github.com/CrunchyData/postgres-operator-client/releases).

### Installing the Client

Once downloaded, move the `kubectl-pgo` binary to `/usr/local/bin` and make it
executable by running the following commands:

```
sudo mv /PATH/TO/kubectl-pgo /usr/local/bin/kubectl-pgo
sudo chmod +x /usr/local/bin/kubectl-pgo
```

### Checking the plugin install

Now that the `kubectl-pgo` binary is installed on your `PATH`, it can be used as
a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/).
Run the following command to ensure that the plugin is working:

```
kubectl pgo version
```

or if running in OpenShift:
```
oc pgo version
```

## Compatibility

The `pgo` CLI supports all actively maintained versions of PGO v5+.

## More Information

For more about PGO, please see the
[PGO Documentation](https://access.crunchydata.com/documentation/postgres-operator/).
