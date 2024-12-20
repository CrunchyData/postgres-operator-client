// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPrimaryInstanceLabels(t *testing.T) {

	assert.Equal(t, PrimaryInstanceLabels("testcluster1"),
		"postgres-operator.crunchydata.com/cluster=testcluster1,"+
			"postgres-operator.crunchydata.com/data=postgres,"+
			"postgres-operator.crunchydata.com/role=master")
}
