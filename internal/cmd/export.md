## The support export command 

### What is it for?

The `support export` command is meant to give users a simple command to run that will produce a tar file of information that they can then use to debug or send to Crunchy Data to debug.

### What are its invariants?

#### Only work on one cluster at a time

If the cluster doesn't exist, exit without creating a tar.

#### Fail gracefully when forbidden from retrieving K8s API info

If we run into an error getting info for part X, move on to part Y, Z, et al. _while_ also surfacing the error.

### What does it do (in order)?

* Check postgrescluster exists (fail hard)
* Create output tar file & defer close
* Gather CLI version (from code)
* Get all postgrescluster names
* Get Kubernetes context (fail hard)
* Get server version (fail hard)
* Gather node info
* Gather namespace info
* Write cluster spec (gotten in step 1)
* Gather namespaced resources for cluster:
  * statefulsets
  * deployments
  * replicasets
  * jobs
  * cronjobs
  * poddisruptionbudgets
  * pods
  * persistentvolumeclaims
  * configmaps
  * services
  * endpoints
  * serviceaccounts
  * ingresses
  * limitranges
* Gather events
* Gather logs
* Gather postgresql logs
* Get monitoring logs
* Gather patroni info
* Gather pgBackRest info
* Gather process info
* Gather system time
* Gather list of kubectl plugins
