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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

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
		// Limit the number of args, that is, only one cluster name
		Args: cobra.ExactArgs(1),
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

	// `backup` command accepts `repoName`, `force-conflicts` and `options` flags;
	// multiple options flags can be used, with each becoming a new line
	// in the options array on the spec
	backup := pgBackRestBackup{
		Config: config,
	}
	cmdBackup.Flags().BoolVar(&backup.ForceConflicts, "force-conflicts", false, "take ownership and overwrite the backup settings")
	cmdBackup.Flags().StringVar(&backup.RepoName, "repoName", "", "repoName to backup to")
	cmdBackup.Flags().StringArrayVar(&backup.Options, "options", []string{},
		"options for taking a backup; can be used multiple times")

	// Define the 'backup' command
	cmdBackup.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		// Pass args[0] as the name of the cluster object, limited to one through `ExactArgs(1)`
		err = backup.Run(client, cmd, args[0])

		if err == nil {
			// Consider a `--wait` flag
			cmd.Printf("%s/%s backup initiated\n", mapping.Resource.Resource, args[0])
		}

		return err
	}

	return cmdBackup
}

type pgBackRestBackup struct {
	Config *internal.Config

	ForceConflicts bool
	Options        []string
	RepoName       string
}

func (backup pgBackRestBackup) modifyIntent(
	intent *unstructured.Unstructured, now time.Time,
) error {
	intent.SetAnnotations(internal.MergeStringMaps(
		intent.GetAnnotations(), map[string]string{
			"postgres-operator.crunchydata.com/pgbackrest-backup": now.UTC().Format(time.RFC3339),
		}))

	if value, path := backup.Options, []string{
		"spec", "backups", "pgbackrest", "manual", "options",
	}; len(value) == 0 {
		unstructured.RemoveNestedField(intent.Object, path...)
	} else if err := unstructured.SetNestedStringSlice(
		intent.Object, value, path...,
	); err != nil {
		return err
	}

	if value, path := backup.RepoName, []string{
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

func (backup pgBackRestBackup) Run(client dynamic.NamespaceableResourceInterface,
	cmd *cobra.Command,
	clusterName string) error {

	var (
		cluster   *unstructured.Unstructured
		err       error
		namespace string
		patch     []byte
	)

	ctx := context.Background()

	// Get the namespace. This will either be from the Kubernetes configuration
	// or from the --namespace (-n) flag.
	if namespace, err = backup.Config.Namespace(); err != nil {
		return err
	}

	if cluster, err = client.Namespace(namespace).Get(ctx,
		clusterName,
		metav1.GetOptions{},
	); err != nil {
		return err
	}

	intent := new(unstructured.Unstructured)
	if err = internal.ExtractFieldsInto(
		cluster, intent, backup.Config.Patch.FieldManager); err != nil {
		return err
	}
	if err = backup.modifyIntent(intent, time.Now()); err != nil {
		return err
	}

	if patch, err = intent.MarshalJSON(); err != nil {
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
	if _, err = client.Namespace(namespace).Patch(ctx,
		clusterName,
		types.ApplyPatchType,
		patch,
		backup.Config.Patch.PatchOptions(patchOptions),
	); err != nil {
		if apierrors.IsConflict(err) {
			cmd.Printf("SUGGESTION: The --force-conflicts flag may help in performing this operation.")
			return err
		}
		cmd.Printf("\nError requesting update: %s\n", err)
		return err
	}

	return err
}
