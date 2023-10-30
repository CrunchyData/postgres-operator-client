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
	"io"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newShowCommand returns the show subcommand of the PGO plugin. The 'show' command
// allows you to display particular details related to the PostgreSQL cluster.
func newShowCommand(config *internal.Config) *cobra.Command {

	cmdShow := &cobra.Command{
		Use:   "show",
		Short: "Show PostgresCluster details",
		Long:  "Show allows you to display particular details related to the PostgresCluster",
	}

	cmdShow.AddCommand(
		newShowBackupCommand(config),
		newShowHACommand(config),
	)

	// No arguments for 'show', but there are arguments for the subcommands, e.g.
	// 'show backup'
	cmdShow.Args = cobra.NoArgs

	return cmdShow
}

// newShowBackupCommand returns the backup subcommand of the show command. The
// 'backup' command displays the output of the 'pgbackrest info' command.
// - https://pgbackrest.org/command.html ('8 Info Command (info)')
func newShowBackupCommand(config *internal.Config) *cobra.Command {

	cmdShowBackup := &cobra.Command{
		Use:     "backup CLUSTER_NAME",
		Aliases: []string{"backups"},
		Short:   "Show backup information for a PostgresCluster",
		Long: `Show backup information for a PostgresCluster from 'pgbackrest info' command.

### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]

### Usage`,
	}

	cmdShowBackup.Example = internal.FormatExample(`# Show every repository of the 'hippo' postgrescluster
pgo show backup hippo

# Show every repository of the 'hippo' postgrescluster as JSON
pgo show backup hippo --output=json

# Show one repository of the 'hippo' postgrescluster
pgo show backup hippo --repoName=repo1

### Example output
stanza: db
    status: ok
    cipher: none

    db (current)
        wal archive min/max (14): 000000010000000000000001/000000010000000000000004

        full backup: 20231023-201416F
            timestamp start/stop: 2023-10-23 20:14:16+00 / 2023-10-23 20:14:32+00
            wal start/stop: 000000010000000000000002 / 000000010000000000000002
            database size: 33.5MB, database backup size: 33.5MB
            repo1: backup set size: 4.2MB, backup size: 4.2MB`)

	// Define the command flags.
	// - https://pgbackrest.org/command.html
	// - output: '8.1.1 Output Option (--output)'
	// - repoName: '8.4.1 Set Repository Option (--repo)'
	var output string
	var repoName string
	cmdShowBackup.Flags().StringVarP(&output, "output", "o", "text",
		"output format. types supported: text,json")
	cmdShowBackup.Flags().StringVar(&repoName, "repoName", "",
		"Set the repository name for the command. example: repo1")

	// Limit the number of args, that is, only one cluster name
	cmdShowBackup.Args = cobra.ExactArgs(1)

	// Define the 'show backup' command
	cmdShowBackup.RunE = func(cmd *cobra.Command, args []string) error {

		// The only thing we need is the value after 'repo' which should be an
		// integer. If anything else is provided, we let the pgbackrest command
		// handle validation.
		repoNum := strings.TrimPrefix(repoName, "repo")

		exec, err := getPrimaryExec(config, args)
		if err != nil {
			return err
		}

		stdout, stderr, err := Executor(exec).pgBackRestInfo(output, repoNum)
		if err != nil {
			return err
		}

		// Print the output received.
		cmd.Printf(stdout)
		if stderr != "" {
			cmd.Printf("\nError returned: %s\n", stderr)
		}

		return nil
	}

	return cmdShowBackup
}

// newShowHACommand returns the output of the 'patronictl list' command.
// - https://patroni.readthedocs.io/en/latest/patronictl.html#patronictl-list
func newShowHACommand(config *internal.Config) *cobra.Command {

	cmdShowHA := &cobra.Command{
		Use:   "ha CLUSTER_NAME",
		Short: "Show 'patronictl list' for a PostgresCluster.",
		Long: `Show 'patronictl list' for a PostgresCluster.

#### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]

### Usage`}

	cmdShowHA.Example = internal.FormatExample(`# Show 'patronictl list' for the 'hippo' postgrescluster
pgo show ha hippo

# Show 'patronictl list' JSON output for the 'hippo' postgrescluster
pgo show ha hippo --json

### Example output
+ Cluster: hippo-ha (7295822780081832000) -----+--------+---------+----+-----------+
| Member          | Host                       | Role   | State   | TL | Lag in MB |
+-----------------+----------------------------+--------+---------+----+-----------+
| hippo-00-cwqq-0 | hippo-00-cwqq-0.hippo-pods | Leader | running |  1 |           |
+-----------------+----------------------------+--------+---------+----+-----------+
	`)

	var json bool
	cmdShowHA.Flags().BoolVar(&json, "json", false, "json format")

	// Limit the number of args, that is, only one cluster name
	cmdShowHA.Args = cobra.ExactArgs(1)

	// Define the 'show backup' command
	cmdShowHA.RunE = func(cmd *cobra.Command, args []string) error {

		exec, err := getPrimaryExec(config, args)
		if err != nil {
			return err
		}

		stdout, stderr, err := Executor(exec).patronictl("list", json)
		if err != nil {
			return err
		}

		// Print the output received.
		cmd.Printf(stdout)
		if stderr != "" {
			cmd.Printf("\nError returned: %s\n", stderr)
		}

		return nil
	}

	return cmdShowHA
}

// getPrimaryExec returns a executor function for the primary Pod to allow for
// commands to be run against it.
func getPrimaryExec(config *internal.Config, args []string) (
	func(stdin io.Reader, stdout io.Writer, stderr io.Writer, command ...string) error,
	error,
) {

	// configure client
	ctx := context.Background()
	rest, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	client, err := corev1.NewForConfig(rest)
	if err != nil {
		return nil, err
	}

	// Get the namespace. This will either be from the Kubernetes configuration
	// or from the --namespace (-n) flag.
	configNamespace, err := config.Namespace()
	if err != nil {
		return nil, err
	}

	// Get the primary instance Pod by its labels. For a Postgres cluster
	// named 'hippo', we'll use the following:
	//    postgres-operator.crunchydata.com/cluster=hippo
	//    postgres-operator.crunchydata.com/data=postgres
	//    postgres-operator.crunchydata.com/role=master
	pods, err := client.Pods(configNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.PrimaryInstanceLabels(args[0]),
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("primary instance Pod not found")
	}

	PodExec, err := util.NewPodExecutor(rest)
	if err != nil {
		return nil, err
	}

	// Create an executor and attempt to get the pgBackRest info output.
	exec := func(stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {
		return PodExec(pods.Items[0].GetNamespace(), pods.Items[0].GetName(),
			util.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	return exec, err
}
