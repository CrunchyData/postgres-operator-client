// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgresoperator

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolvePostgresClusterVersion(t *testing.T) {
	t.Run("ExplicitOverrideWins", func(t *testing.T) {
		// Set env var to a different version to make sure override wins.
		t.Setenv(EnvAPIVersion, APIVersionV1Beta1)
		got, err := ResolvePostgresClusterVersion(APIVersionV1, nil)
		assert.NilError(t, err)
		assert.Equal(t, got, APIVersionV1)
	})

	t.Run("InvalidOverride", func(t *testing.T) {
		_, err := ResolvePostgresClusterVersion("v2", nil)
		assert.ErrorContains(t, err, "unsupported PostgresCluster API version")
	})

	t.Run("EnvVarUsedWhenOverrideEmpty", func(t *testing.T) {
		t.Setenv(EnvAPIVersion, APIVersionV1)
		got, err := ResolvePostgresClusterVersion("", nil)
		assert.NilError(t, err)
		assert.Equal(t, got, APIVersionV1)
	})

	t.Run("InvalidEnvVar", func(t *testing.T) {
		t.Setenv(EnvAPIVersion, "v2")
		_, err := ResolvePostgresClusterVersion("", nil)
		assert.ErrorContains(t, err, "PGO_API_VERSION")
	})

	t.Run("DefaultWhenNoOverrideNoEnvNoDiscovery", func(t *testing.T) {
		t.Setenv(EnvAPIVersion, "")
		got, err := ResolvePostgresClusterVersion("", nil)
		assert.NilError(t, err)
		assert.Equal(t, got, DefaultAPIVersion)
		assert.Equal(t, DefaultAPIVersion, APIVersionV1Beta1)
	})
}

func TestValidateAPIVersion(t *testing.T) {
	assert.NilError(t, validateAPIVersion(APIVersionV1))
	assert.NilError(t, validateAPIVersion(APIVersionV1Beta1))
	assert.ErrorContains(t, validateAPIVersion(""), "unsupported")
	assert.ErrorContains(t, validateAPIVersion("v2"), "unsupported")
}

func TestGroupVersion(t *testing.T) {
	assert.Equal(t, GroupVersion(APIVersionV1), "postgres-operator.crunchydata.com/v1")
	assert.Equal(t, GroupVersion(APIVersionV1Beta1), "postgres-operator.crunchydata.com/v1beta1")
}
