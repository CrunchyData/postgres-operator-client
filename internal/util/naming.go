// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

	// LabelMonitoring is used to identify monitoring Pods.
	// Older versions of PGO monitoring use the label 'postgres-operator-monitoring'.
	LabelMonitoring = "app.kubernetes.io/name in (postgres-operator-monitoring,crunchy-monitoring)"

	// LabelOperator is used to identify operator Pods
	LabelOperator = "postgres-operator.crunchydata.com/control-plane"

	// LabelPGBackRestDedicated is used to identify the Repo Host pod
	LabelPGBackRestDedicated = labelPrefix + "pgbackrest-dedicated"
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

// DBInstanceLabels provides labels for a PostgreSQL cluster primary or replica instance
func DBInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataPostgres
}

// PrimaryInstanceLabels provides labels for a PostgreSQL cluster primary instance
func PrimaryInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataPostgres + "," +
		LabelRole + "=" + RolePatroniLeader
}

// RepoHostInstanceLabels provides labels for a Backrest Repo Host instances
func RepoHostInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelPGBackRestDedicated + "="
}

// PostgresUserSecretLabels provides labels for the Postgres user Secret
func PostgresUserSecretLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelRole + "=" + RolePostgresUser
}

// AllowUpgradeAnnotation is the annotation key to allow of PostgresCluster
// to upgrade. Its value is the name of the PGUpgrade object.
func AllowUpgradeAnnotation() string {
	return labelPrefix + "allow-upgrade"
}
