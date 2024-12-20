// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmp

import (
	"strings"

	gotest "gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

type Comparison = gotest.Comparison

// MarshalMatches converts actual to YAML and compares that to expected.
func MarshalMatches(actual interface{}, expected string) Comparison {
	b, err := yaml.Marshal(actual)
	if err != nil {
		return func() gotest.Result { return gotest.ResultFromError(err) }
	}
	return gotest.DeepEqual(string(b), strings.Trim(expected, "\t\n")+"\n")
}
