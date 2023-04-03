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
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/fatih/color"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
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
		newShowStatusCommand(config),
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

#### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]`,
	}

	cmdShowBackup.Example = internal.FormatExample(`
# Show every repository of the 'hippo' postgrescluster
pgo show backup hippo

# Show every repository of the 'hippo' postgrescluster as JSON
pgo show backup hippo --output=json

# Show one repository of the 'hippo' postgrescluster
pgo show backup hippo --repoName=repo1
	`)

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

		// configure client
		ctx := context.Background()
		rest, err := config.ToRESTConfig()
		if err != nil {
			return err
		}
		client, err := corev1.NewForConfig(rest)
		if err != nil {
			return err
		}

		// Get the namespace. This will either be from the Kubernetes configuration
		// or from the --namespace (-n) flag.
		configNamespace, err := config.Namespace()
		if err != nil {
			return err
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
			return err
		}

		if len(pods.Items) != 1 {
			return fmt.Errorf("Primary instance Pod not found.")
		}

		PodExec, err := util.NewPodExecutor(rest)
		if err != nil {
			return err
		}

		// Create an executor and attempt to get the pgBackRest info output.
		exec := func(stdin io.Reader, stdout, stderr io.Writer,
			command ...string) error {
			return PodExec(pods.Items[0].GetNamespace(), pods.Items[0].GetName(),
				util.ContainerDatabase, stdin, stdout, stderr, command...)
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

// newShowStatusCommand returns the status subcommand of the show command. The
// 'status' reports if a cluster or a set of clusters are up and running.
func newShowStatusCommand(config *internal.Config) *cobra.Command {

	cmdShowStatus := &cobra.Command{
		Use:   "status CLUSTER_NAME",
		Short: "Show status for a PostgresCluster. If none specified, returns status for all clusters.",
		Long: `Show status information (shutdown/running) for a PostgresCluster.

#### RBAC Requirements
    Resources          Verbs
    ---------          -----
    postgrescluster    [list]
`,
	}

	cmdShowStatus.Example = internal.FormatExample(`
# Show status of the 'hippo' postgrescluster
pgo show status hippo -n flamingo

# Show status of all postgresclusters
pgo show status

# Show status of all postgresclusters in a given namespace
pgo show status -n flamingo
	`)

	// Define the 'show backup' command
	cmdShowStatus.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		_, clientCrunchy, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		configFlags := genericclioptions.NewConfigFlags(false)
		ns := configFlags.Namespace

		clusters, err := clientCrunchy.Namespace(*ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			os.Exit(1)
		}
		for _, cluster := range clusters.Items {
			displayStatusOf(cluster)
		}

		return nil
	}

	return cmdShowStatus
}

func displayStatusOf(pgCluster unstructured.Unstructured) {
	green := color.New(color.FgGreen)
	white := color.New(color.FgWhite)
	red := color.New(color.FgRed)
	clName := getPresentationNameForCluster(pgCluster)
	_, _ = green.Printf("%s", clName)
	_, _ = white.Printf(" %s [", strings.Repeat(".", 60-len(clName)+20))
	spec := pgCluster.Object["spec"].(map[string]interface{})
	switch spec["shutdown"] {
	case false:
		_, _ = green.Printf("Running")
	default:
		_, _ = red.Printf("Shutdown")
	}
	_, _ = white.Printf("]")
	fmt.Println("")
}
