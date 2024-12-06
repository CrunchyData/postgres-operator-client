// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
}

func (cfg *Config) Namespace() (string, error) {
	ns, _, err := cfg.ToRawKubeConfigLoader().Namespace()
	return ns, err
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
