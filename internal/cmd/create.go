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

package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newCreateCommand returns the create subcommand of the PGO plugin.
// Subcommands of create will be use to create objects, backups, etc.
func newCreateCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a resource",
		Long:  "Create a resource",
	}

	cmd.AddCommand(newCreateClusterCommand(config))

	return cmd
}

// newCreateClusterCommand returns the create cluster subcommand.
// create cluster will take a cluster name as an argument and create a basic
// cluster using a kube client
func newCreateClusterCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "postgrescluster CLUSTER_NAME",
		Aliases: []string{"postgresclusters"},
		Short:   "Create PostgresCluster with a given name",
		Long: `Create basic PostgresCluster with a given name.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [create]

### Usage`,
	}

	cmd.Args = cobra.ExactArgs(1)

	var pgMajorVersion int
	cmd.Flags().IntVar(&pgMajorVersion, "pg-major-version", 0, "Set the Postgres major version")
	cobra.CheckErr(cmd.MarkFlagRequired("pg-major-version"))

	var backupsDisabled bool
	cmd.Flags().BoolVar(&backupsDisabled, "disable-backups", false, "Disable backups")

	cmd.Example = internal.FormatExample(`# Create a postgrescluster with Postgres 15
pgo create postgrescluster hippo --pg-major-version 15

# Create a postgrescluster with backups disabled (only available in CPK v5.7+)
# Requires confirmation
pgo create postgrescluster hippo --disable-backups

### Example output	
postgresclusters/hippo created`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		clusterName := args[0]

		namespace, err := config.Namespace()
		if err != nil {
			return err
		}

		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		cluster, err := generateUnstructuredClusterYaml(clusterName, strconv.Itoa(pgMajorVersion))
		if err != nil {
			return err
		}

		if backupsDisabled {
			fmt.Print("WARNING: Running a production postgrescluster without backups " +
				"is not recommended. \nAre you sure you want " +
				"to continue without backups? (yes/no): ")
			var confirmed *bool
			for i := 0; confirmed == nil && i < 10; i++ {
				// retry 10 times or until a confirmation is given or denied,
				// whichever comes first
				confirmed = util.Confirm(os.Stdin, os.Stdout)
			}

			if confirmed == nil || !*confirmed {
				return nil
			}

			unstructured.RemoveNestedField(cluster.Object, "spec", "backups")
		}

		u, err := client.
			Namespace(namespace).
			Create(ctx, cluster, config.Patch.CreateOptions(metav1.CreateOptions{}))
		if err != nil {
			return err
		}

		cmd.Printf("%s/%s created\n", mapping.Resource.Resource, u.GetName())

		return nil
	}

	return cmd
}

// generateUnstructuredClusterYaml takes a name and returns a PostgresCluster
// in the unstructured format.
func generateUnstructuredClusterYaml(name, pgMajorVersion string) (*unstructured.Unstructured, error) {
	var cluster unstructured.Unstructured
	err := yaml.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: %s
spec:
  postgresVersion: %s
  instances:
  - dataVolumeClaimSpec:
      accessModes:
      - "ReadWriteOnce"
      resources:
        requests:
          storage: 1Gi
  backups:
    pgbackrest:
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
`, name, pgMajorVersion)), &cluster)

	if err != nil {
		return nil, err
	}

	return &cluster, nil
}
