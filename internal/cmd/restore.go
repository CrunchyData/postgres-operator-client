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
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
)

func newRestoreCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore CLUSTER_NAME",
		Short: "Restore cluster",
		Long: `Restore the data of a PostgreSQL cluster from a backup either by
using the current "spec.backups.pgbackrest.restore" settings on the PostgreSQL
cluster or by using flags to write your settings. Overwriting those settings
may require the --force-conflicts flags.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]
	
### Usage`,
	}

	cmd.Example = internal.FormatExample(`# Restore the 'hippo' cluster using the latest backup and replay all available WAL
pgo restore hippo --repoName repo1

# Restore the 'hippo' cluster to a specific point in time
pgo restore hippo --repoName repo1 --options '--type=time --target="2021-06-09 14:15:11-04"'

### Example output
WARNING: You are about to restore from pgBackRest with {options:[] repoName:repo1}
WARNING: This action is destructive and PostgreSQL will be unavailable while its data is restored.

Do you want to continue? (yes/no): yes
postgresclusters/hippo patched

# Resolve ownership conflict
pgo restore hippo --force-conflicts
`)

	restore := pgBackRestRestore{Config: config}

	cmd.Flags().StringArrayVar(&restore.Options, "options", nil,
		`options to pass to the "pgbackrest restore" command; can be used multiple times`)

	cmd.Flags().StringVar(&restore.RepoName, "repoName", "",
		"repository to restore from")

	cmd.Flags().BoolVar(&restore.ForceConflicts, "force-conflicts", false, "take ownership and overwrite the restore settings")

	// Only one positional argument: the PostgresCluster name.
	cmd.Args = cobra.ExactArgs(1)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		restore.PostgresCluster = strings.TrimPrefix(args[0], "postgrescluster/")
		if strings.HasPrefix(args[0], "postgresclusters/") {
			restore.PostgresCluster = strings.TrimPrefix(args[0], "postgresclusters/")
		}

		return restore.Run(context.Background())
	}

	cmd.AddCommand(newRestoreDisableCommand(config))

	return cmd
}

func newRestoreDisableCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable CLUSTER_NAME",
		Short: "Disable restores for a PostgresCluster",
		Long: `Update a PostgresCluster spec to disable restores.

This is recommended after your restore is complete. Running "pgo restore" will enable restores again.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]

### Usage`,
	}

	disable := pgBackRestRestoreDisable{Config: config}

	// Only one positional argument: the PostgresCluster name.
	cmd.Args = cobra.ExactArgs(1)

	cmd.Example = internal.FormatExample(`# Disable the restore section on the 'hippo' cluster
pgo restore disable hippo

### Example output
postgresclusters/hippo patched`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		disable.PostgresCluster = args[0]
		return disable.Run(context.Background())
	}

	return cmd
}

type pgBackRestRestore struct {
	*internal.Config

	Options        []string
	RepoName       string
	ForceConflicts bool

	PostgresCluster string
}

func (config pgBackRestRestore) Run(ctx context.Context) error {
	details := func(cluster *unstructured.Unstructured) (out struct {
		options  []string
		repoName string
	}) {
		restore, _, _ := unstructured.NestedMap(
			cluster.Object, "spec", "backups", "pgbackrest", "restore")

		out.options, _, _ = unstructured.NestedStringSlice(restore, "options")
		out.repoName, _, _ = unstructured.NestedString(restore, "repoName")

		return
	}

	mapping, client, err := v1beta1.NewPostgresClusterClient(config)
	if err != nil {
		return err
	}

	namespace, err := config.Namespace()
	if err != nil {
		return err
	}

	// Fetch the cluster to (1) see if it exists and (2) extract CLI managed fields.
	cluster, err := client.Namespace(namespace).Get(ctx,
		config.PostgresCluster, metav1.GetOptions{})
	if err != nil {
		return err
	}

	intent := new(unstructured.Unstructured)
	if err := internal.ExtractFieldsInto(cluster, intent, config.Patch.FieldManager); err != nil {
		return err
	}
	if err := config.modifyIntent(intent, time.Now()); err != nil {
		return err
	}

	patch, err := intent.MarshalJSON()
	if err != nil {
		return err
	}
	patchOptions := metav1.PatchOptions{
		DryRun: []string{metav1.DryRunAll},
	}

	if config.ForceConflicts {
		b := true
		patchOptions.Force = &b
	}

	// Perform a dry-run patch to understand what settings will be used should
	// the restore proceed.
	cluster, err = client.Namespace(namespace).Patch(ctx,
		config.PostgresCluster, types.ApplyPatchType, patch,
		config.Patch.PatchOptions(patchOptions))
	if err != nil {
		if apierrors.IsConflict(err) {
			fmt.Fprintf(config.Out, "SUGGESTION: The --force-conflicts flag may help in performing this operation.\n")
		}
		return err
	}

	fmt.Fprintf(config.Out,
		"WARNING: You are about to restore from pgBackRest with %+v\n"+
			"WARNING: This action is destructive and PostgreSQL will be"+
			" unavailable while its data is restored.\n\n"+
			"Do you want to continue? (yes/no): ",
		details(cluster))

	if confirmed := config.confirm(5); confirmed == nil || !*confirmed {
		return nil
	}

	patchOptions = metav1.PatchOptions{}
	if config.ForceConflicts {
		b := true
		patchOptions.Force = &b
	}

	// They agreed to continue. Send the patch again without dry-run.
	_, err = client.Namespace(namespace).Patch(ctx,
		config.PostgresCluster, types.ApplyPatchType, patch,
		config.Patch.PatchOptions(patchOptions))

	if err == nil {
		fmt.Fprintf(config.Out, "%s/%s patched\n",
			mapping.Resource.Resource, config.PostgresCluster)
	}

	return err
}

func (config pgBackRestRestore) confirm(attempts int) *bool {
	for i := 0; i < attempts; i++ {
		if confirmed := confirm(config.In, config.Out); confirmed != nil {
			return confirmed
		}
	}

	return nil
}

func (config pgBackRestRestore) modifyIntent(
	intent *unstructured.Unstructured, now time.Time,
) error {
	intent.SetAnnotations(internal.MergeStringMaps(
		intent.GetAnnotations(), map[string]string{
			"postgres-operator.crunchydata.com/pgbackrest-restore": now.UTC().Format(time.RFC3339),
		}))

	if err := unstructured.SetNestedField(intent.Object, true,
		"spec", "backups", "pgbackrest", "restore", "enabled",
	); err != nil {
		return err
	}

	if value, path := config.Options, []string{
		"spec", "backups", "pgbackrest", "restore", "options",
	}; len(value) == 0 {
		unstructured.RemoveNestedField(intent.Object, path...)
	} else if err := unstructured.SetNestedStringSlice(
		intent.Object, value, path...,
	); err != nil {
		return err
	}

	if value, path := config.RepoName, []string{
		"spec", "backups", "pgbackrest", "restore", "repoName",
	}; len(value) == 0 {
		unstructured.RemoveNestedField(intent.Object, path...)
	} else if err := unstructured.SetNestedField(
		intent.Object, value, path...,
	); err != nil {
		return err
	}

	return nil
}

type pgBackRestRestoreDisable struct {
	*internal.Config

	PostgresCluster string
}

func (config pgBackRestRestoreDisable) Run(ctx context.Context) error {
	mapping, client, err := v1beta1.NewPostgresClusterClient(config)
	if err != nil {
		return err
	}

	namespace, err := config.Namespace()
	if err != nil {
		return err
	}

	// Fetch the cluster to (1) see if it exists and (2) extract CLI managed fields.
	cluster, err := client.Namespace(namespace).Get(ctx,
		config.PostgresCluster, metav1.GetOptions{})
	if err != nil {
		return err
	}

	intent := new(unstructured.Unstructured)
	if err := internal.ExtractFieldsInto(cluster, intent, config.Patch.FieldManager); err != nil {
		return err
	}
	if err := config.modifyIntent(intent); err != nil {
		return err
	}

	patch, err := intent.MarshalJSON()

	if err == nil {
		_, err = client.Namespace(namespace).Patch(ctx,
			config.PostgresCluster, types.ApplyPatchType, patch,
			config.Patch.PatchOptions(metav1.PatchOptions{}))
	}

	if err == nil {
		fmt.Fprintf(config.Out, "%s/%s patched\n",
			mapping.Resource.Resource, config.PostgresCluster)
	}

	return err
}

func (config pgBackRestRestoreDisable) modifyIntent(
	intent *unstructured.Unstructured,
) error {
	unstructured.RemoveNestedField(intent.Object,
		"spec", "backups", "pgbackrest", "restore")

	internal.RemoveEmptySections(intent,
		"spec", "backups", "pgbackrest")

	return nil
}
