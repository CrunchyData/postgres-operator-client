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

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
)

type ShutdownRequestArgs struct {
	ClusterName      string
	Config           *internal.Config
	ForceConflicts   bool
	Namespace        string
	NewShutdownValue bool
	Mapping          *meta.RESTMapping
}

func newStartCommand(config *internal.Config) *cobra.Command {
	cmdStart := &cobra.Command{
		Use:   "start CLUSTER_NAME",
		Short: "Start cluster",
		Long: `Start sets the spec.shutdown field to false, allowing you to start a PostgreSQL cluster.
The --force-conflicts flag may be required if the spec.shutdown field has been updated by another client.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    postgresclusters.postgres-operator.crunchydata.com  [get patch]

### Usage`,
	}
	cmdStart.Example = internal.FormatExample(`# Start a 'hippo' postgrescluster.
pgo start hippo

# Resolve ownership conflict
pgo start hippo --force-conflicts

### Example output
postgresclusters/hippo start initiated`)

	// Limit the number of args, that is, only one cluster name
	cmdStart.Args = cobra.ExactArgs(1)

	var forceConflicts bool
	cmdStart.Flags().BoolVar(&forceConflicts, "force-conflicts", false, "take ownership and overwrite the shutdown setting")
	cmdStart.RunE = func(cmd *cobra.Command, args []string) error {
		mapping, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}
		namespace, err := config.Namespace()
		if err != nil {
			return err
		}
		requestArgs := ShutdownRequestArgs{
			ClusterName:      args[0],
			Config:           config,
			ForceConflicts:   forceConflicts,
			Namespace:        namespace,
			NewShutdownValue: false,
			Mapping:          mapping,
		}

		cluster, err := getPostgresCluster(client, requestArgs)
		if err != nil {
			return err
		}

		msg, err := patchClusterShutdown(cluster, client, requestArgs)
		if msg != "" {
			cmd.Printf(msg)
		}
		if err != nil {
			return err
		}
		return nil
	}

	return cmdStart
}

func patchClusterShutdown(cluster *unstructured.Unstructured, client dynamic.NamespaceableResourceInterface, args ShutdownRequestArgs) (string, error) {
	ctx := context.Background()

	currShutdownVal, found, err := unstructured.NestedBool(cluster.Object, "spec", "shutdown")
	if err != nil {
		return "", err
	}
	// If the shutdown status is equal to the intent of the command, do nothing.
	if found && currShutdownVal == args.NewShutdownValue {
		// If NewShutdownValue == true, we intend to stop the cluster.
		if args.NewShutdownValue {
			return "Cluster already Stopped. Nothing to do.\n", nil
		}
		return "Cluster already Started. Nothing to do.\n", nil
	}

	// Construct the payload.
	intent := new(unstructured.Unstructured)
	if err := internal.ExtractFieldsInto(cluster, intent, args.Config.Patch.FieldManager); err != nil {
		return "", err
	}
	if err := unstructured.SetNestedField(intent.Object, args.NewShutdownValue, "spec", "shutdown"); err != nil {
		return "", err
	}
	patch, err := intent.MarshalJSON()
	if err != nil {
		return "", err
	}
	patchOptions := metav1.PatchOptions{}
	if args.ForceConflicts {
		b := true
		patchOptions.Force = &b
	}

	// Patch the update.
	_, err = client.Namespace(args.Namespace).Patch(ctx,
		args.ClusterName, types.ApplyPatchType, patch,
		args.Config.Patch.PatchOptions(patchOptions))
	if err != nil {
		if apierrors.IsConflict(err) {
			return "SUGGESTION: The --force-conflicts flag may help in performing this operation.\n", err
		}
		return "", err
	}
	var initiatedMsg string
	// If NewShutdownValue == true, we intend to stop the cluster.
	if args.NewShutdownValue {
		initiatedMsg = "stop initiated"
	} else {
		initiatedMsg = "start initiated"
	}
	return fmt.Sprintf("%s/%s %s\n", args.Mapping.Resource.Resource, args.ClusterName, initiatedMsg), err
}

func getPostgresCluster(client dynamic.NamespaceableResourceInterface, args ShutdownRequestArgs) (*unstructured.Unstructured, error) {
	ctx := context.Background()
	cluster, err := client.Namespace(args.Namespace).Get(ctx,
		args.ClusterName, metav1.GetOptions{})
	return cluster, err
}
