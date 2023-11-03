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
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/yaml"

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
		if stdout, stderr, err := showBackup(config, args, "text", ""); err != nil {
			return err
		} else {
			cmd.Printf(stdout)
			if stderr != "" {
				cmd.Printf("\nError returned: %s\n", stderr)
			}
		}

		// Print the patronictl list output received.
		cmd.Printf("\nHA\n\n")
		if stdout, stderr, err := showHA(config, args, "pretty"); err != nil {
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

		stdout, stderr, err := showBackup(config, args, outputEnum.String(), repoNum)

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

// showBackup execs into the primary Pod, runs the 'pgbackrest info' command and
// returns the command output and/or error
func showBackup(
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

		stdout, stderr, err := showHA(config, args, outputEnum.String())

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

// showHA execs into the primary Pod, runs the 'patronictl list' command and
// returns the command output and/or error
func showHA(
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
		Use:   "user CLUSTER_NAME",
		Short: "Show pguser Secret details for a PostgresCluster.",
		Long: `Show pguser Secret details for a PostgresCluster.

#### RBAC Requirements
    Resources  Verbs
    ---------  -----
    secrets       [list]

### Usage`}

	cmdShowUser.Example = internal.FormatExample(`# Show non-sensitive contents of 'pguser' Secret
pgo show user hippo

# Show contents of 'pguser' Secret, including sensitive fields
pgo show user hippo --show-sensitive-fields

### Example output
pgo show user hippo
SECRET: hippo-pguser-hippo
  DBNAME: hippo
  HOST: hippo-primary.postgres-operator.svc
  PORT: 5432
  USER: hippo
	`)

	var fields bool
	cmdShowUser.Flags().BoolVarP(&fields, "show-sensitive-fields", "f", false, "show sensitive user fields")

	// Limit the number of args, that is, only one cluster name
	cmdShowUser.Args = cobra.ExactArgs(1)

	// Define the 'show backup' command
	cmdShowUser.RunE = func(cmd *cobra.Command, args []string) error {

		stdout, err := showUser(config, args, fields)
		if err != nil {
			return err
		}

		cmd.Print(stdout)

		return nil
	}

	return cmdShowUser
}

// showUser returns a string with the decoded contents of the cluster's user Secrets.
func showUser(config *internal.Config, args []string, showSensitive bool) (string, error) {

	// break out keys based on whether sensitive information is included
	var fields = []string{"dbname", "host", "pgbouncer-host", "pgbouncer-port", "port", "user"}
	var sensitive = []string{"jdbc-uri", "password", "pgbouncer-jdbc-uri", "pgbouncer-uri", "uri", "verifier"}

	if showSensitive {
		fields = append(fields, sensitive...)

		fmt.Print("WARNING: This command will show sensitive password information." +
			"\nAre you sure you want to continue? (yes/no): ")

		var confirmed *bool
		for i := 0; confirmed == nil && i < 10; i++ {
			// retry 10 times or until a confirmation is given or denied,
			// whichever comes first
			confirmed = util.Confirm(os.Stdin, os.Stdout)
		}

		if confirmed == nil || !*confirmed {
			return "", nil
		}
	}

	// configure client
	ctx := context.Background()
	rest, err := config.ToRESTConfig()
	if err != nil {
		return "", err
	}
	client, err := v1.NewForConfig(rest)
	if err != nil {
		return "", err
	}

	// Get the namespace. This will either be from the Kubernetes configuration
	// or from the --namespace (-n) flag.
	configNamespace, err := config.Namespace()
	if err != nil {
		return "", err
	}

	list, err := client.Secrets(configNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.PostgresUserSecretLabels(args[0]),
	})
	if err != nil {
		return "", err
	}

	return userData(fields, list)
}

// userData returns the requested user data from the provided Secret List.
// If the Secret List is empty, return a message stating that.
func userData(fields []string, list *corev1.SecretList) (string, error) {

	var output string

	if len(list.Items) == 0 {
		output += fmt.Sprintln("No user Secrets found.")
	}

	for _, secret := range list.Items {
		output += fmt.Sprintf("SECRET: %s\n", secret.Name)

		// sort keys
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// decode and print keys and values from Secret
		for _, k := range keys {
			b, err := yaml.Marshal(secret.Data[k])
			if err != nil {
				return output, err
			}
			d := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
			_, err = base64.StdEncoding.Decode(d, b)
			if err != nil {
				return output, err
			}
			if slices.Contains(fields, k) {
				output += fmt.Sprintf("  %s: %s\n", strings.ToUpper(k), string(d))
			}
		}
	}
	return output, nil
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
