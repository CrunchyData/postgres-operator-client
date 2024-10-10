// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/crunchydata/postgres-operator-client/internal/cmd"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-pgo", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewPGOCommand(os.Stdin, os.Stdout, os.Stderr)
	cobra.CheckErr(root.Execute())
}
