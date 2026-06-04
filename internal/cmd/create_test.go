// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	postgresoperator "github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com"
	"github.com/crunchydata/postgres-operator-client/internal/testing/cmp"
)

func TestGenerateUnstructuredYaml(t *testing.T) {
	const tmpl = `
apiVersion: postgres-operator.crunchydata.com/%s
kind: PostgresCluster
metadata:
  name: hippo
spec:
  backups:
    pgbackrest:
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - ReadWriteOnce
            resources:
              requests:
                storage: 1Gi
  instances:
  - dataVolumeClaimSpec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
  postgresVersion: 15
`

	// Use literal version strings for the expectation so a rename of one of
	// the postgresoperator constants doesn't quietly continue passing while
	// the actual rendered apiVersion changes.
	for _, tt := range []struct {
		name            string
		apiVersion      string
		expectAPIString string
	}{
		{name: "v1beta1", apiVersion: postgresoperator.APIVersionV1Beta1, expectAPIString: "v1beta1"},
		{name: "v1", apiVersion: postgresoperator.APIVersionV1, expectAPIString: "v1"},
		{name: "empty falls back to default", apiVersion: "", expectAPIString: "v1beta1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, err := generateUnstructuredClusterYaml("hippo", "15", tt.apiVersion)
			assert.NilError(t, err)
			assert.Assert(t, cmp.MarshalMatches(
				interface{}(u),
				fmt.Sprintf(tmpl, tt.expectAPIString),
			))
		})
	}
}
