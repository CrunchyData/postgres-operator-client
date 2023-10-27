// Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
)

// newBackupCommand returns the backup command of the PGO plugin.
// It optionally takes a `repoName` and `options` flag, which it uses
// to update the spec.
func newBackupCommand(config *internal.Config) *cobra.Command {

	cmdBackup := &cobra.Command{
		Use:   "backup CLUSTER_NAME",
		Short: "Backup cluster",
		Long: `Backup allows you to backup a PostgreSQL cluster either by using
the current "spec.backups.pgbackrest.manual" settings on the PostgreSQL cluster
or by using flags to write your settings. Overwriting those settings may require
the --force-conflicts flag.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]

### Usage`,
	}

	cmdBackup.Example = internal.FormatExample(`# Trigger a backup on the 'hippo' postgrescluster using the current spec options
# Note: "spec.backups.pgbackrest.manual.repoName" has to exist for the backup to begin
pgo backup hippo

# Update the 'backups.pgbackrest.manual.repoName' and 'backups.pgbackrest.manual.options' fields
# on the 'hippo' postgrescluster and trigger a backup
pgo backup hippo --repoName="repo1" --options="--type=full"

# Resolve ownership conflict
pgo backup hippo --force-conflicts

### Example output
postgresclusters/hippo backup initiated`)

	// Limit the number of args, that is, only one cluster name
	cmdBackup.Args = cobra.ExactArgs(1)

	// `backup` command accepts `repoName`, `force-conflicts` and `options` flags;
	// multiple options flags can be used, with each becoming a new line
	// in the options array on the spec
	backup := pgBackRestBackup{}
	cmdBackup.Flags().BoolVar(&backup.ForceConflicts, "force-conflicts", false, "take ownership and overwrite the backup settings")
	cmdBackup.Flags().StringVar(&backup.RepoName, "repoName", "", "repoName to backup to")
	cmdBackup.Flags().StringArrayVar(&backup.Options, "options", []string{},
		"options for taking a backup; can be used multiple times")

	// Define the 'backup' command
	cmdBackup.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		ctx := context.Background()
		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		// Get the namespace. This will either be from the Kubernetes configuration
		// or from the --namespace (-n) flag.
		configNamespace, err := config.Namespace()
		if err != nil {
			return err
		}

		cluster, err := client.Namespace(configNamespace).Get(ctx,
			args[0], // the name of the cluster object, limited to one name through `ExactArgs(1)`
			metav1.GetOptions{},
		)
		if err != nil {
			return err
		}

		intent := new(unstructured.Unstructured)
		if err := internal.ExtractFieldsInto(cluster, intent, config.Patch.FieldManager); err != nil {
			return err
		}
		if err := backup.modifyIntent(intent, time.Now()); err != nil {
			return err
		}

		patch, err := intent.MarshalJSON()
		if err != nil {
			cmd.Printf("\nError packaging payload: %s\n", err)
			return err
		}

		// Update the spec/annotate
		// TODO(benjaminjb): Would we want to allow a dry-run option here?
		patchOptions := metav1.PatchOptions{}
		if backup.ForceConflicts {
			b := true
			patchOptions.Force = &b
		}
		_, err = client.Namespace(configNamespace).Patch(ctx,
			args[0], // the name of the cluster object, limited to one name through `ExactArgs(1)`
			types.ApplyPatchType,
			patch,
			config.Patch.PatchOptions(patchOptions),
		)

		if err != nil {
			cmd.Printf("\nError requesting update: %s\n", err)
			return err
		}

		// Print the output received.
		// TODO(benjaminjb): consider a more informative output
		cmd.Printf("%s/%s backup initiated\n", mapping.Resource.Resource, args[0])

		return nil
	}

	return cmdBackup
}

type pgBackRestBackup struct {
	Options        []string
	RepoName       string
	ForceConflicts bool
}

func (config pgBackRestBackup) modifyIntent(
	intent *unstructured.Unstructured, now time.Time,
) error {
	intent.SetAnnotations(internal.MergeStringMaps(
		intent.GetAnnotations(), map[string]string{
			"postgres-operator.crunchydata.com/pgbackrest-backup": now.UTC().Format(time.RFC3339),
		}))

	if value, path := config.Options, []string{
		"spec", "backups", "pgbackrest", "manual", "options",
	}; len(value) == 0 {
		unstructured.RemoveNestedField(intent.Object, path...)
	} else if err := unstructured.SetNestedStringSlice(
		intent.Object, value, path...,
	); err != nil {
		return err
	}

	if value, path := config.RepoName, []string{
		"spec", "backups", "pgbackrest", "manual", "repoName",
	}; len(value) == 0 {
		unstructured.RemoveNestedField(intent.Object, path...)
	} else if err := unstructured.SetNestedField(
		intent.Object, value, path...,
	); err != nil {
		return err
	}

	internal.RemoveEmptySections(intent,
		"spec", "backups", "pgbackrest", "manual")

	return nil
}
