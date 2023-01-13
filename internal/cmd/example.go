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
	"bytes"
	"context"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/dynamic"
)

// newExampleCommand returns the example subcommand of the PGO plugin.
// - https://github.com/spf13/cobra/blob/-/user_guide.md
//
//nolint:deadcode,unused
func newExampleCommand(kubeconfig *genericclioptions.ConfigFlags) *cobra.Command {
	// NOTE: Take as arguments 👆 anything you want passed in.

	cmd := &cobra.Command{
		Use:   "example",
		Short: "short description",
		Long:  "long description",
	}

	// NOTE: Instantiate anything you need here.
	var region string

	// NOTE: Define flags and their behavior here. (optional)
	cmd.Flags().StringVarP(&region, "region", "", "", "flag description")
	cobra.CheckErr(cmd.MarkFlagRequired("region"))

	// NOTE: Define validation for positional arguments here. (optional)
	cmd.Args = cobra.NoArgs

	// NOTE: Define what happens when this command is called. (optional)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()

		cmd.Println("region:", region)

		config, err := kubeconfig.ToRESTConfig()
		if err != nil {
			return err
		}

		client, err := dynamic.NewForConfig(config)
		if err != nil {
			return err
		}

		list, err := client.Resource(schema.GroupVersionResource{
			Version: "v1", Resource: "namespaces",
		}).List(ctx, metav1.ListOptions{
			Limit: 10,
		})
		if err != nil {
			return err
		}

		var buf bytes.Buffer

		p := printers.NewTablePrinter(printers.PrintOptions{})
		if err := p.PrintObj(list, &buf); err != nil {
			return err
		}

		cmd.Println(buf.String())

		return nil
	}

	return cmd
}
