// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

func newStopCommand(config *internal.Config) *cobra.Command {
	cmdStop := &cobra.Command{
		Use:   "stop CLUSTER_NAME",
		Short: "Stop cluster",
		Long: `Stop sets the spec.shutdown field to true, allowing you to stop a PostgreSQL cluster.
The --force-conflicts flag may be required if the spec.shutdown field has been used before.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]

### Usage`,
	}
	cmdStop.Example = internal.FormatExample(`# Stop a 'hippo' postgrescluster.
pgo stop hippo

# Resolve ownership conflict
pgo stop hippo --force-conflicts

### Example output
postgresclusters/hippo stop initiated`)

	// Limit the number of args, that is, only one cluster name
	cmdStop.Args = cobra.ExactArgs(1)

	var forceConflicts bool
	cmdStop.Flags().BoolVar(&forceConflicts, "force-conflicts", false, "take ownership and overwrite the shutdown setting")
	cmdStop.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Print("WARNING: Stopping a postgrescluster is not destructive but " +
			"it will take your database offline until you restart it. \nAre you sure you want " +
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

		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}
		namespace, err := config.Namespace()
		if err != nil {
			return err
		}

		requestArgs := ShutdownRequestArgs{
			ClusterName:      args[0],
			Config:           config,
			ForceConflicts:   forceConflicts,
			Namespace:        namespace,
			NewShutdownValue: true,
			Mapping:          mapping,
		}
		cluster, err := getPostgresCluster(client, requestArgs)
		if err != nil {
			return err
		}

		msg, err := patchClusterShutdown(cluster, client, requestArgs)
		if msg != "" {
			cmd.Printf("%s", msg)
		}
		if err != nil {
			return err
		}
		return nil
	}

	return cmdStop
}
