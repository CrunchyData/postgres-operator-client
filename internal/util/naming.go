// Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
)

const (
	// Data values

	// DataPostgres is a LabelData value that indicates the object has PostgreSQL data.
	DataPostgres = "postgres"
)

const (
	// Role values

	// RolePatroniLeader is the LabelRole that Patroni sets on the Pod that is
	// currently the leader.
	RolePatroniLeader = "master"
)

const (
	// Container names

	// ContainerDatabase is the name of the container running PostgreSQL and
	// supporting tools: Patroni, pgBackRest, etc.
	ContainerDatabase = "database"
)

// PrimaryInstanceLabels provides labels for a PostgreSQL cluster primary instance
func PrimaryInstanceLabels(clusterName string) string {
	return LabelCluster + "=" + clusterName + "," +
		LabelData + "=" + DataPostgres + "," +
		LabelRole + "=" + RolePatroniLeader
}
