// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator-client/internal/testing/cmp"
)

func TestGenerateUnstructuredYaml(t *testing.T) {
	expect := `
apiVersion: postgres-operator.crunchydata.com/v1beta1
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

	u, err := generateUnstructuredClusterYaml("hippo", "15")
	assert.NilError(t, err)

	assert.Assert(t, cmp.MarshalMatches(
		interface{}(u),
		expect,
	))

}
