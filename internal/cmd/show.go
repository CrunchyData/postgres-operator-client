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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newShowCommand returns the show subcommand of the PGO plugin. The 'show' command
// allows you to display particular details related to the PostgreSQL cluster.
func newShowCommand(config *internal.Config) *cobra.Command {

	cmdShow := &cobra.Command{
		Use:   "show",
		Short: "Show PostgresCluster details",
		Long: `Show allows you to display particular details related to the PostgresCluster.

### RBAC Requirements
    Resources  Verbs
    ---------  -----
    pods       [list]
    pods/exec  [create]

### Usage`,
	}

	cmdShow.Example = internal.FormatExample(`# Show the backup and HA output of the 'hippo' postgrescluster
pgo show hippo

### Example output
BACKUP

stanza: db
    status: ok
    cipher: none

    db (current)
        wal archive min/max (14): 000000010000000000000001/000000010000000000000003

        full backup: 20231030-183841F
            timestamp start/stop: 2023-10-30 18:38:41+00 / 2023-10-30 18:38:46+00
            wal start/stop: 000000010000000000000002 / 000000010000000000000002
            database size: 25.3MB, database backup size: 25.3MB
            repo1: backup set size: 3.2MB, backup size: 3.2MB

HA

+ Cluster: hippo-ha (7295822780081832000) -----+--------+---------+----+-----------+
| Member          | Host                       | Role   | State   | TL | Lag in MB |
+-----------------+----------------------------+--------+---------+----+-----------+
| hippo-00-cwqq-0 | hippo-00-cwqq-0.hippo-pods | Leader | running |  1 |           |
+-----------------+----------------------------+--------+---------+----+-----------+
`)

	cmdShow.AddCommand(
		newShowBackupCommand(config),
		newShowHACommand(config),
		newShowUserCommand(config),
	)

	// Limit the number of args, that is, only one cluster name
	cmdShow.Args = cobra.ExactArgs(1)

	// Define the 'show backup' command
	cmdShow.RunE = func(cmd *cobra.Command, args []string) error {

		// Print the pgbackrest info output received.
		cmd.Printf("BACKUP\n\n")
		if stdout, stderr, err := getBackup(config, args, "text", ""); err != nil {
			return err
		} else {
			cmd.Printf(stdout)
			if stderr != "" {
				cmd.Printf("\nError returned: %s\n", stderr)
			}
		}

		// Print the patronictl list output received.
		cmd.Printf("\nHA\n\n")
		if stdout, stderr, err := getHA(config, args, "pretty"); err != nil {
			return err
		} else {
			cmd.Printf(stdout)
			if stderr != "" {
				cmd.Printf("\nError returned: %s\n", stderr)
			}
		}
		return nil
	}

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
	var repoName string
	var outputEnum = util.TextPGBackRest
	cmdShowBackup.Flags().VarP(&outputEnum, "output", "o",
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

		stdout, stderr, err := getBackup(config, args, outputEnum.String(), repoNum)

		if err == nil {
			cmd.Printf(stdout)
			if stderr != "" {
				cmd.Printf("\nError returned: %s\n", stderr)
			}
		}

		return err
	}

	return cmdShowBackup
}

// getBackup execs into the primary Pod, runs the 'pgbackrest info' command and
// returns the command output and/or error
func getBackup(
	config *internal.Config,
	args []string,
	output string,
	repoNum string) (string, string, error) {

	exec, err := getPrimaryExec(config, args)
	if err != nil {
		return "", "", err
	}

	return Executor(exec).pgBackRestInfo(output, repoNum)
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
pgo show ha hippo --output json

### Example output
+ Cluster: hippo-ha (7295822780081832000) -----+--------+---------+----+-----------+
| Member          | Host                       | Role   | State   | TL | Lag in MB |
+-----------------+----------------------------+--------+---------+----+-----------+
| hippo-00-cwqq-0 | hippo-00-cwqq-0.hippo-pods | Leader | running |  1 |           |
+-----------------+----------------------------+--------+---------+----+-----------+
	`)

	var outputEnum = util.PrettyPatroni
	cmdShowHA.Flags().VarP(&outputEnum, "output", "o",
		"output format. types supported: pretty,tsv,json,yaml")

	// Limit the number of args, that is, only one cluster name
	cmdShowHA.Args = cobra.ExactArgs(1)

	// Define the 'show backup' command
	cmdShowHA.RunE = func(cmd *cobra.Command, args []string) error {

		stdout, stderr, err := getHA(config, args, outputEnum.String())

		if err == nil {
			cmd.Printf(stdout)
			if stderr != "" {
				cmd.Printf("\nError returned: %s\n", stderr)
			}
		}

		return err
	}

	return cmdShowHA
}

// getHA execs into the primary Pod, runs the 'patronictl list' command and
// returns the command output and/or error
func getHA(
	config *internal.Config,
	args []string,
	output string) (string, string, error) {
	exec, err := getPrimaryExec(config, args)
	if err != nil {
		return "", "", err
	}

	return Executor(exec).patronictl("list", output)
}

// newShowUserCommand returns the decoded contents of the cluster's user Secrets.
func newShowUserCommand(config *internal.Config) *cobra.Command {

	cmdShowUser := &cobra.Command{
		Use:   "user USER_NAME --cluster CLUSTER_NAME",
		Short: "Show details for a PostgresCluster user.",
		Long: `Show details for a PostgresCluster user. Only shows
details for the default user for a PostgresCluster
or for users defined on the PostgresCluster spec.
Use the "--show-connection-info" flag to get the
connection info, including password.

#### RBAC Requirements
    Resources  Verbs
    ---------  -----
    secrets       [list]

### Usage`}

	cmdShowUser.Example = internal.FormatExample(`# Show non-sensitive contents of users for "hippo" cluster
pgo show user --cluster hippo

# Show non-sensitive contents of user "rhino" for "hippo" cluster
pgo show user rhino --cluster hippo

# Show connection info for user "rhino" for "hippo" cluster,
# including sensitive password info
pgo show user rhino --cluster hippo --show-connection-info

### Example output
# Showing all the users of the "hippo" cluster
CLUSTER  USERNAME
hippo    hippo
hippo    rhino

# Showing the connection info for user "hippo" of cluster "hippo"
WARNING: This command will show sensitive password information.
Are you sure you want to continue? (yes/no): yes

Connection information for hippo for hippo cluster
Connection info string:
    dbname=hippo host=hippo-primary.postgres-operator.svc port=5432 user=hippo password=<password>
Connection URL:
    postgres://<password>@hippo-primary.postgres-operator.svc:5432/hippo`)

	var fields bool
	cmdShowUser.Flags().BoolVar(&fields, "show-connection-info", false, "show sensitive user fields")

	var cluster string
	cmdShowUser.Flags().StringVarP(&cluster, "cluster", "c", "", "Set the Postgres cluster name (required)")
	cobra.CheckErr(cmdShowUser.MarkFlagRequired("cluster"))

	// Limit the number of args to at most one pguser name
	cmdShowUser.Args = cobra.MaximumNArgs(1)

	// Define the 'show backup' command
	cmdShowUser.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		rest, err := config.ToRESTConfig()
		if err != nil {
			return err
		}
		client, err := v1.NewForConfig(rest)
		if err != nil {
			return err
		}

		secretList, err := getUsers(client, config, cluster, args)
		if err != nil {
			return err
		}

		// If no user info found, exit early
		if len(secretList.Items) == 0 {
			notFoundMessage := "No user information found for cluster " + cluster
			if len(args) > 0 {
				notFoundMessage = notFoundMessage + " / user " + args[0]
			}
			cmd.Print(notFoundMessage + "\n")
			return nil
		}

		// If user info found, print
		return printUsers(cmd, secretList, fields, cluster)
	}

	return cmdShowUser
}

// getUsers returns a string with the decoded contents of the cluster's users' Secrets.
func getUsers(client *v1.CoreV1Client,
	config *internal.Config,
	cluster string,
	args []string,
) (*corev1.SecretList, error) {
	ctx := context.Background()

	// Get the namespace. This will either be from the Kubernetes configuration
	// or from the --namespace (-n) flag.
	configNamespace, err := config.Namespace()
	if err != nil {
		return nil, err
	}

	// Set up the labels for listing the secrets; add the user label is present in args
	labelSelector := util.PostgresUserSecretLabels(cluster)
	if len(args) > 0 {
		labelSelector = labelSelector +
			",postgres-operator.crunchydata.com/pguser=" + args[0]
	}

	return client.Secrets(configNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func printUsers(cmd *cobra.Command,
	secretList *corev1.SecretList,
	showSensitive bool,
	clusterName string,
) error {

	// If the user is asking for connection strings, we use the alternate printer.
	if showSensitive {
		return printUserConnectionStrings(cmd, secretList, clusterName)
	}

	// Set up a tabwriter that writes to stdout
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 10, 2, 2, ' ', 0)
	if _, err := writer.Write([]byte("\nCLUSTER\tUSERNAME\n")); err != nil {
		return err
	}
	for _, secret := range secretList.Items {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "%s\t%s\n", clusterName, string(secret.Data["user"]))
		if _, err := writer.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}

	return writer.Flush()
}

func printUserConnectionStrings(cmd *cobra.Command,
	secretList *corev1.SecretList,
	clusterName string,
) error {
	fmt.Print("WARNING: This command will show sensitive password information." +
		"\nAre you sure you want to continue? (yes/no): ")

	var confirmed *bool
	for i := 0; confirmed == nil && i < 10; i++ {
		// retry 10 times or until a confirmation is given or denied,
		// whichever comes first
		confirmed = util.Confirm(os.Stdin, os.Stdout)
	}

	if confirmed == nil || !*confirmed {
		return nil
	}

	cmd.Println()
	for _, secret := range secretList.Items {
		cmd.Printf("Connection information for %s for %s cluster\n", string(secret.Data["user"]), clusterName)
		cmd.Println("Connection info string:")
		dbname := string(secret.Data["user"])
		if dbnameSet, ok := secret.Data["dbname"]; ok {
			dbname = string(dbnameSet)
		}
		cmd.Println("    dbname=" + dbname +
			" host=" + string(secret.Data["host"]) +
			" port=" + string(secret.Data["port"]) +
			" user=" + string(secret.Data["user"]) +
			" password=" + string(secret.Data["password"]))
		cmd.Println("Connection URL:")
		cmd.Println("    postgres://" + string(secret.Data["user"]) +
			":" + string(secret.Data["password"]) +
			"@" + string(secret.Data["host"]) +
			":" + string(secret.Data["port"]) +
			"/" + dbname)
		if uri, ok := secret.Data["pgbouncer-uri"]; ok {
			cmd.Println("PgBouncer connection URL:")
			cmd.Println("    " + string(uri))

			cmd.Println("JDBC PgBouncer connection URL:")
			cmd.Println("    " + string(secret.Data["pgbouncer-jdbc-uri"]))
		}
		cmd.Println()
	}

	return nil
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
	client, err := v1.NewForConfig(rest)
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
