---
title: "pgo, the Postgres Operator Client from Crunchy Data"
date:
draft: false
---

# `pgo`, the Postgres Operator Client from Crunchy Data

 <img width="25%" src="logos/pgo.svg" alt="pgo: The Postgres Operator Client from Crunchy Data" />

Latest Release: {{< param clientVersion >}}

# Install `pgo`, the Postgres Operator for Kubernetes Client

The following steps will allow you to download and install the `pgo` [kubectl plugin][] in your
local environment.

## Prerequisites

Depending on your deployment type, Kubernetes or OpenShift, `kubectl` or `oc` must be installed and
configured in your environment. For the purposes of these instructions we will be using the `kubectl`
client. The `pgo` [kubectl plugin][] will use the role-based access controls (RBAC) that are
configured for your `kubectl` client.

## Download the binary

The `kubectl-pgo` binary is available either through the Crunchy Data [Access Portal][] or via [GitHub][].

## Installing the Client

Once downloaded, move the `kubectl-pgo` binary to `/usr/local/bin` and make it executable by running
the following commands:

```
sudo mv /PATH/TO/kubectl-pgo /usr/local/bin/kubectl-pgo
sudo chmod +x /usr/local/bin/kubectl-pgo
```

## Checking the plugin install

Now that the `kubectl-pgo` binary is installed on your `PATH`, it can be used as a [kubectl plugin][].

Run the following command to ensure that the plugin is working:

```
kubectl pgo version
```

or if running in OpenShift:
```
oc pgo version
```

[kubectl plugin]: https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
[Access Portal]: https://access.crunchydata.com/downloads/
[GitHub]: https://github.com/CrunchyData/postgres-operator-client/releases
