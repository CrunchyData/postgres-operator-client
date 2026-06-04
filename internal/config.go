// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Config struct {
	*genericclioptions.ConfigFlags
	genericclioptions.IOStreams

	Patch PatchConfig

	// APIVersion is the PostgresCluster API version override (e.g. "v1" or
	// "v1beta1"). When empty, the postgresoperator package falls back to the
	// PGO_API_VERSION env var, then discovery, then a default of v1beta1.
	APIVersion string
}

func (cfg *Config) Namespace() (string, error) {
	ns, _, err := cfg.ToRawKubeConfigLoader().Namespace()
	return ns, err
}

// AddPGOFlags registers PGO-specific persistent flags on the given flag set.
// Currently this adds --pgo-api-version. The flag is bound with an empty
// default so that env-var lookup and discovery happen in a single place
// (postgresoperator.ResolvePostgresClusterVersion); that keeps error
// messages and behavior consistent regardless of whether the value came
// from the flag, the env var, or discovery.
func (cfg *Config) AddPGOFlags(flags *pflag.FlagSet) {
	flags.StringVar(&cfg.APIVersion, "pgo-api-version", "",
		"PostgresCluster API version to use (v1 or v1beta1). "+
			"Defaults to PGO_API_VERSION env var, then auto-detect, then v1beta1.")
}

type PatchConfig struct {
	FieldManager string
}

func (cfg *PatchConfig) AddFlags(flags *pflag.FlagSet) {
	// See [k8s.io/kubectl/pkg/cmd/util.AddFieldManagerFlagVar]
	flags.StringVar(&cfg.FieldManager, "field-manager", cfg.FieldManager,
		"Name of the manager used to track field ownership.")
}

// CreateOptions returns a copy of opts with fields set according to cfg.
func (cfg *PatchConfig) CreateOptions(opts metav1.CreateOptions) metav1.CreateOptions {
	opts.FieldManager = cfg.FieldManager
	return opts
}

// PatchOptions returns a copy of opts with fields set according to cfg.
func (cfg *PatchConfig) PatchOptions(opts metav1.PatchOptions) metav1.PatchOptions {
	opts.FieldManager = cfg.FieldManager
	return opts
}
