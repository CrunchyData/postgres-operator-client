// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

// newCreateCommand returns the create subcommand of the PGO plugin.
// Subcommands of create will be use to create objects, backups, etc.
func newCreateCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a resource",
		Long:  "Create a resource",
	}

	cmd.AddCommand(newCreateClusterCommand(config))
	cmd.AddCommand(newCreateOperatorCommand(config))

	return cmd
}

// newCreateClusterCommand returns the create cluster subcommand.
// create cluster will take a cluster name as an argument and create a basic
// cluster using a kube client
func newCreateClusterCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "postgrescluster CLUSTER_NAME",
		Aliases: []string{"postgresclusters"},
		Short:   "Create PostgresCluster with a given name",
		Long: `Create basic PostgresCluster with a given name.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [create]

### Usage`,
	}

	cmd.Args = cobra.ExactArgs(1)

	var pgMajorVersion int
	cmd.Flags().IntVar(&pgMajorVersion, "pg-major-version", 0, "Set the Postgres major version")
	cobra.CheckErr(cmd.MarkFlagRequired("pg-major-version"))

	var backupsDisabled bool
	cmd.Flags().BoolVar(&backupsDisabled, "disable-backups", false, "Disable backups")

	cmd.Example = internal.FormatExample(`# Create a postgrescluster with Postgres 15
pgo create postgrescluster hippo --pg-major-version 15

# Create a postgrescluster with backups disabled (only available in CPK v5.7+)
# Requires confirmation
pgo create postgrescluster hippo --disable-backups

### Example output	
postgresclusters/hippo created`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		clusterName := args[0]

		namespace, err := config.Namespace()
		if err != nil {
			return err
		}

		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		cluster, err := generateUnstructuredClusterYaml(clusterName, strconv.Itoa(pgMajorVersion))
		if err != nil {
			return err
		}

		if backupsDisabled {
			fmt.Print("WARNING: Running a production postgrescluster without backups " +
				"is not recommended. \nAre you sure you want " +
				"to continue without backups? (yes/no): ")
			var confirmed *bool
			for i := 0; confirmed == nil && i < 10; i++ {
				// retry 10 times or until a confirmation is given or denied,
				// whichever comes first
				confirmed = util.Confirm(os.Stdin, os.Stdout)
			}

			if confirmed == nil || !*confirmed {
				return nil
			}

			unstructured.RemoveNestedField(cluster.Object, "spec", "backups")
		}

		u, err := client.
			Namespace(namespace).
			Create(ctx, cluster, config.Patch.CreateOptions(metav1.CreateOptions{}))
		if err != nil {
			return err
		}

		cmd.Printf("%s/%s created\n", mapping.Resource.Resource, u.GetName())

		return nil
	}

	return cmd
}

// generateUnstructuredClusterYaml takes a name and returns a PostgresCluster
// in the unstructured format.
func generateUnstructuredClusterYaml(name, pgMajorVersion string) (*unstructured.Unstructured, error) {
	var cluster unstructured.Unstructured
	err := yaml.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: %s
spec:
  postgresVersion: %s
  instances:
  - dataVolumeClaimSpec:
      accessModes:
      - "ReadWriteOnce"
      resources:
        requests:
          storage: 1Gi
  backups:
    pgbackrest:
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
`, name, pgMajorVersion)), &cluster)

	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// Creating a custom value type that satisfies the Value interface
// https://pkg.go.dev/github.com/spf13/pflag#Value
// This allows us to set a custom enum-type var
type installToolString string

const (
	installToolStringKustomize installToolString = "kustomize"
	installToolStringHelm      installToolString = "helm"
)

func (its *installToolString) String() string {
	return string(*its)
}

func (its *installToolString) Set(v string) error {
	switch v {
	case "kustomize", "helm":
		*its = installToolString(v)
		return nil
	default:
		return errors.New(`must be one of "kustomize" or "helm"`)
	}
}

// Type is only used in help text
func (its *installToolString) Type() string {
	return "installToolString"
}

// newCreateOperatorCommand returns the create operator subcommand.
// create operator will take a cluster name as an argument and create a basic
// operator using a kube client.
func newCreateOperatorCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"operators"},
		Short:   "Create Postgres-Operator",
		Long: `Create Postgres-Operator.

An operator is deployed:

- from any remote or local source (defaults to github.com/CrunchyData/postgres-operator-examples for kustomize installations and oci://registry.developers.crunchydata.com/crunchydata/pgo for Helm installations);
- using either kustomize or Helm (defaults to kustomize).

Note: The tool used to install must already be present.

If Helm is used as the deployment tool, you may optionally supply a name (defaults to "crunchy") and values file (defaults to "values.yaml" file).

### RBAC Requirements
    Resources	Verbs
    ---------	-----
    pods	[create]
	TODO...

### Usage`,
	}

	cmd.Args = cobra.ExactArgs(0)

	var source string
	cmd.Flags().StringVar(&source,
		"source",
		"https://github.com/CrunchyData/postgres-operator-examples.git/kustomize/install/default?timeout=120&ref=main",
		"Source to deploy the operator from; defaults to github.com/CrunchyData/postgres-operator-examples for kustomize installations and oci://registry.developers.crunchydata.com/crunchydata/pgo for Helm installations")

	// Default to using kustomize
	var installTool = installToolStringKustomize
	cmd.Flags().VarP(&installTool,
		"install-tool",
		"t",
		"Tool to deploy the operator (either kustomize or helm); defaults to kustomize")

	var installName string
	cmd.Flags().StringVar(&installName,
		"name",
		"crunchy",
		"Name for the Helm installation (defaults to 'crunchy')")

	var valueFile string
	cmd.Flags().StringVar(&valueFile,
		"values",
		"",
		"Location of a values file")

	cmd.Example = internal.FormatExample(`# Create an operator with defaults:
pgo create operator

# Create an operator with Helm in a particular namespace.
# The namespace has to exist prior to the command being run:
pgo create operator --install-tool helm --namespace postgres-operator

# Create an operator...
`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {

		// Note: Kustomize has a namespace in the overlay; for now,
		// use that namespace.
		installNamespace, err := config.Namespace()
		if err != nil {
			return err
		}

		var ex *exec.Cmd
		// Need to install with --server-side flag due to size
		// of CRD
		if installTool == installToolStringKustomize {
			ex = exec.Command("kubectl", "apply", "--kustomize", source, "--server-side")
		}

		if installTool == installToolStringHelm {
			// If the source wasn't changed from default, set it for our Helm default
			if !cmd.Flags().Changed("source") {
				source = "oci://registry.developers.crunchydata.com/crunchydata/pgo"
			}
			args := []string{"install", installName, source, "--namespace", installNamespace}
			if valueFile != "" {
				args = append(args, "--values", valueFile)
			}
			ex = exec.Command("helm", args...)
		}

		msg, err := ex.Output()
		if err != nil {
			return err
		}

		cmd.Printf("%s", msg)

		return nil
	}

	return cmd
}
