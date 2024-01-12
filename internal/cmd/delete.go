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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newDeleteCommand returns the delete subcommand of the PGO plugin.
// Subcommands of delete will be use to delete objects, etc.
func newDeleteCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource",
		Long:  "Delete a resource",
	}

	cmd.AddCommand(newDeleteClusterCommand(config))

	return cmd
}

// newDeleteClusterCommand returns the delete cluster subcommand.
// delete cluster will take a cluster name as an argument
func newDeleteClusterCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postgrescluster CLUSTER_NAME",
		Short: "Delete a PostgresCluster",
		Long: `Delete a PostgresCluster with a given name.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [delete]

### Usage`,
	}

	cmd.Args = cobra.ExactArgs(1)

	cmd.Example = internal.FormatExample(`# Delete a postgrescluster
pgo delete postgrescluster hippo

### Example output	
WARNING: Deleting a postgrescluster is destructive and data retention is dependent on PV configuration. 
Are you sure you want to continue? (yes/no): yes
postgresclusters/hippo deleted`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		clusterName := args[0]

		fmt.Print("WARNING: Deleting a postgrescluster is destructive and data " +
			"retention is dependent on PV configuration. \nAre you sure you want " +
			"to continue? (yes/no): ")
		var confirmed *bool
		for i := 0; confirmed == nil && i < 10; i++ {
			// retry 10 times or until a confirmation is given or denied,
			// whichever comes first
			confirmed = util.Confirm(os.Stdin, os.Stdout)
		}

		if confirmed == nil || !*confirmed {
			return nil
		}

		namespace, err := config.Namespace()
		if err != nil {
			return err
		}

		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		err = client.
			Namespace(namespace).
			Delete(ctx, clusterName, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		cmd.Printf("%s/%s deleted\n", mapping.Resource.Resource, clusterName)

		return nil
	}

	return cmd
}
