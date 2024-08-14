// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

const (
	// Labels

	// labelPrefix is the prefix common to all PostgresCluster object labels.
	labelPrefix = "postgres-operator.crunchydata.com/"

	// LabelCluster is used to label PostgresCluster objects.
	LabelCluster = labelPrefix + "cluster"

	// LabelData is used to identify Pods and Volumes store Postgres data.
	LabelData = labelPrefix + "data"

	// LabelRole is used to identify object roles.
	LabelRole = labelPrefix + "role"

	// LabelMonitoring is used to identify monitoring Pods
	LabelMonitoring = "app.kubernetes.io/name=postgres-operator-monitoring"

	// LabelOperator is used to identify operator Pods
	LabelOperator = "postgres-operator.crunchydata.com/control-plane"
)

const (
	// Data values

	// DataPostgres is a LabelData value that indicates the object has PostgreSQL data.
	DataPostgres = "postgres"

	// DataBackrest is a LabelData value that indicate the object is a Repo Host.
	DataBackrest = "pgbackrest"
)

const (
	// Role values

	// RolePatroniLeader is the LabelRole that Patroni sets on the Pod that is
	// currently the leader.
	RolePatroniLeader = "master"

	// RolePatroniReplica is the LabelRole that Patroni sets on the Pod that is
	// currently a replica.
	RolePatroniReplica = "replica"

	// RolePostgresUser is the LabelRole applied to PostgreSQL user secrets.
	RolePostgresUser = "pguser"
)

const (
	// Container names

	// ContainerDatabase is the name of the container running PostgreSQL and
	// supporting tools: Patroni, pgBackRest, etc.
	ContainerDatabase = "database"

	ContainerPGBackrest = "pgbackrest"
)

// PrimaryInstanceLabels provides labels for a PostgreSQL cluster primary instance
func PrimaryInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataPostgres + "," +
		LabelRole + "=" + RolePatroniLeader
}

// ReplicaInstanceLabels provides labels for a PostgreSQL cluster replica instances
func ReplicaInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataPostgres + "," +
		LabelRole + "=" + RolePatroniReplica
}

// RepoHostInstanceLabels provides labels for a Backrest Repo Host instances
func RepoHostInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataBackrest
}

// PostgresUserSecretLabels provides labels for the Postgres user Secret
func PostgresUserSecretLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelRole + "=" + RolePostgresUser
}
