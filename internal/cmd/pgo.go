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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/crunchydata/postgres-operator-client/internal"
)

// NewPGOCommand returns the root command of the PGO plugin. This command
// prints the same information as its --help flag: the available subcommands
// and their short descriptions.
func NewPGOCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	config := &internal.Config{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   genericclioptions.IOStreams{In: stdin, Out: stdout, ErrOut: stderr},
		Patch:       internal.PatchConfig{FieldManager: filepath.Base(os.Args[0])},
	}

	root := &cobra.Command{
		// When this executable is named `kubectl-pgo`, it can be invoked as
		// either `kubectl-pgo` or `kubectl pgo`. Assume the former for now.
		// - https://docs.k8s.io/tasks/extend-kubectl/kubectl-plugins/
		//
		// NOTE: The cobra package notices spaces in this value. The "first word"
		// appears in various places as the [cobra.Command.Name].
		Use: "kubectl-pgo",

		Short: "pgo is a kubectl plugin for PGO, the open source Postgres Operator",
		Long: strings.TrimSpace(`
pgo is a kubectl plugin for PGO, the open source Postgres Operator from Crunchy Data.

	https://github.com/CrunchyData/postgres-operator
`),

		// Support shell completion, but do not list it as one of the available
		// subcommands. It can be invoked via `kubectl pgo completion`.
		//
		// NOTE: `kubectl` completion does not currently consider plugins.
		// - https://issue.k8s.io/74178
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},

		// Do not append "[flags]" to the UseLine.
		DisableFlagsInUseLine: true,

		// Do not include a heading with the date on which documentation was generated.
		DisableAutoGenTag: true,

		// Print the long description and usage when there is no subcommand.
		Run: nil,
	}

	// set the default directory for flag ----cache-dir
	cfgDir := "$HOME/.kube/cache"
	config.CacheDir = &cfgDir

	// Add flags for kubeconfig, authentication, namespace, and timeout to
	// every subcommand.
	// - https://docs.k8s.io/concepts/configuration/organize-cluster-access-kubeconfig/
	config.ConfigFlags.AddFlags(root.PersistentFlags())

	// Defined command output. If not set, it falls back to [os.Stderr].
	// - https://pkg.go.dev/github.com/spf13/cobra#Command.Print
	root.SetOut(stdout)

	root.AddCommand(newBackupCommand(config))
	root.AddCommand(newCreateCommand(config))
	root.AddCommand(newDeleteCommand(config))
	root.AddCommand(newShowCommand(config))

	return root
}
