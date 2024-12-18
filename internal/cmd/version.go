// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	v1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator-client/internal"
)

// newVersionCommand returns the CLI client version and the Postgres operator
// version.
func newVersionCommand(config *internal.Config) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "version",
		Short: "PGO client and operator versions",
		Long: `Version displays the versions of the PGO client and the Crunchy Postgres Operator

### RBAC Requirements
    Resources                                       Verbs
    ---------                                       -----
    customresourcedefinitions.apiextensions.k8s.io  [get]

    Note: This RBAC needs to be cluster-scoped.

### Usage`,
	}

	// No arguments for 'version'
	cmd.Args = cobra.NoArgs

	var clientOnly bool
	cmd.Flags().BoolVar(&clientOnly, "client", false, "If true, shows client version only (no server required).")

	cmd.Example = internal.FormatExample(fmt.Sprintf(`# Request the version of the client and the operator
pgo version

### Example output
Client Version: %s
Operator Version: v5.7.0`, clientVersion))

	cmd.RunE = func(cmd *cobra.Command, args []string) error {

		cmd.Printf("Client Version: %s\n", clientVersion)
		if clientOnly {
			return nil
		}

		ctx := context.Background()
		restConfig, err := config.ToRESTConfig()
		if err != nil {
			return err
		}
		// get a client capable of retrieving the PostgresCluster CRD
		client, err := v1.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		crd, err := client.CustomResourceDefinitions().
			Get(ctx, "postgresclusters.postgres-operator.crunchydata.com", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if crd != nil &&
			crd.ObjectMeta.Labels != nil &&
			crd.ObjectMeta.Labels["app.kubernetes.io/version"] != "" {

			cmd.Printf("Operator Version: v%s\n", crd.ObjectMeta.Labels["app.kubernetes.io/version"])
		} else {
			cmd.Println("Operator version not found.")
		}

		return nil
	}

	return cmd
}
