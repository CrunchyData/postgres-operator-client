// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/crunchydata/postgres-operator-client/internal"
)

// newSupportCommand returns the support subcommand of the PGO plugin.
func newSupportCommand(config *internal.Config) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "support",
		Short: "Crunchy Support commands for PGO",
	}

	cmd.AddCommand(newSupportExportCommand(config))

	return cmd
}
