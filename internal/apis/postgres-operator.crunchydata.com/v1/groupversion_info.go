// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains API Schema definitions.
package v1

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	GroupVersion = schema.GroupVersion{Group: "postgres-operator.crunchydata.com", Version: "v1"}
)
