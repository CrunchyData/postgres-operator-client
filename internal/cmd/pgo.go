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
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdPGO(streams genericclioptions.IOStreams) *cobra.Command {
	options := genericclioptions.NewConfigFlags(true)

	root := &cobra.Command{
		Use:   "kubectl pgo",
		Short: "pgo is a kubectl plugin for PGO, the open source Postgres Operator",
		Long: strings.TrimSpace(`
pgo is a kubectl plugin for PGO, the open source Postgres Operator from Crunchy Data.

	https://github.com/CrunchyData/postgres-operator
`),

		// Print the long description and usage when there is no subcommand.
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	options.AddFlags(root.PersistentFlags())

	return root
}
