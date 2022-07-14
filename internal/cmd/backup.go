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

package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newBackupCommand returns the backup command of the PGO plugin.
// It optionally takes a `repoName` and `options` flag, which it uses
// to update the spec; if left out, the backup command will use whatever
// is extant on the spec.
func newBackupCommand(kubeconfig *genericclioptions.ConfigFlags) *cobra.Command {

	cmdBackup := &cobra.Command{
		Use:   "backup",
		Short: "Backup cluster",
		Long:  "Backup allows you to take a backup of a PostgreSQL cluster",
	}

	cmdBackup.Example = `  # Trigger a backup on the 'hippo' pod using the current spec options
  pgo backup hippo

  # Update the 'backups.pgbackrest.manual.repoName' and 'backups.pgbackrest.manual.options' fields
  # on the 'hippo' postgrescluster and trigger a backup
  pgo backup hippo --repoName="repo1"  --options="--type=full"
`

	// Limit the number of args, that is, only one cluster name
	cmdBackup.Args = cobra.ExactArgs(1)

	// `backup` command accepts `repoName` and `options` flags with the following notes:
	// 1) multiple options flags can be used, with each becoming a new line
	// in the options array on the spec
	// 2) the `repoName` and `options` flags must be used together
	var repoName string
	var options []string
	cmdBackup.Flags().StringVar(&repoName, "repoName", "", "repoName to backup to")
	cmdBackup.Flags().StringArrayVar(&options, "options", []string{},
		"options for taking a backup; can be used multiple times")
	cmdBackup.MarkFlagsRequiredTogether("repoName", "options")

	// Define the 'backup' command
	// TODO(benjaminjb): Refactor when we have a postgrescluster client util
	cmdBackup.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		ctx := context.Background()
		config, err := kubeconfig.ToRESTConfig()
		if err != nil {
			return err
		}
		client, err := dynamic.NewForConfig(config)
		if err != nil {
			return err
		}

		// Get the namespace. This will either be from the Kubernetes configuration
		// or from the --namespace (-n) flag.
		configNamespace, _, err := kubeconfig.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}

		patch, err := generateBackupPatch(time.Now().Format(time.Stamp),
			repoName, options)
		if err != nil {
			cmd.Printf("\nError packaging payload: %s\n", err)
			return err
		}

		// Update the spec/annotate
		// TODO(benjaminjb): Would we want to allow a dry-run option here?
		// TODO(benjaminjb): Would we want to allow a force option here?
		_, err = client.Resource(schema.GroupVersionResource{
			Group:    util.PostgresGroup,
			Version:  util.PostgresVersion,
			Resource: util.PostgresResource,
		}).Namespace(configNamespace).Patch(ctx,
			args[0], // the name of the cluster object, limited to one name through `ExactArgs(1)`
			types.MergePatchType,
			patch,
			// TODO(benjaminjb): What do we want the FieldManager to be?
			metav1.PatchOptions{
				FieldManager: "postgrescluster-cli",
			})

		if err != nil {
			cmd.Printf("\nError requesting update: %s\n", err)
			return err
		}

		// Print the output received.
		// TODO(benjaminjb): consider a more informative output
		cmd.Printf("postgresclusters/%s backup initiated\n", args[0])

		return nil
	}

	return cmdBackup
}

// generateBackupPatch takes a trigger (string) to add to the postgrescluster annotations;
// it takes repoName and options from the CLI flags (optional); if the flags are omitted,
// the backup patch will just be the trigger annotation.
// For ease of legibility and output (e.g., to make sure each entry in `options` is a separate
// line in the spec), this creates a golang struct and marshals it as json
func generateBackupPatch(trigger, repoName string, options []string) (out []byte, err error) {
	update := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"postgres-operator.crunchydata.com/pgbackrest-backup": trigger,
			},
		},
	}

	if repoName != "" {
		update["spec"] = map[string]interface{}{
			"backups": map[string]interface{}{
				"pgbackrest": map[string]interface{}{
					"manual": map[string]interface{}{
						"repoName": repoName,
						"options":  options,
					},
				},
			},
		}
	}

	return json.Marshal(update)
}
