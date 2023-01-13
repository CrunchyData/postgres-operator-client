// Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
  postgresVersion: 14
`

	u, err := generateUnstructuredClusterYaml("hippo")
	assert.NilError(t, err)

	assert.Assert(t, cmp.MarshalMatches(
		interface{}(u),
		expect,
	))

}
