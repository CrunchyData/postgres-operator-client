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
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator-client/internal/testing/cmp"
)

func TestPGBackRestBackupModifyIntent(t *testing.T) {
	now := time.Date(2020, 4, 5, 6, 7, 8, 99, time.FixedZone("ZONE", -11))

	for _, tt := range []struct {
		Name, Before, After string
		Backup              pgBackRestBackup
	}{
		{
			Name: "Zero",
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
			`),
		},
		{
			Name: "Options",
			Backup: pgBackRestBackup{
				Options: []string{"--quoth=raven --midnight=dreary", "--ever=never"},
			},
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      manual:
        options:
        - --quoth=raven --midnight=dreary
        - --ever=never
			`),
		},
		{
			Name:   "RepoName",
			Backup: pgBackRestBackup{RepoName: "testRepo"},
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      manual:
        repoName: testRepo
			`),
		},
		{
			Name: "OldRepoAndOptions",
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: existingTrigger
spec:
  backups:
    pgbackrest:
      manual:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
			`),
		},
		{
			Name:   "NewRepoButOptions",
			Backup: pgBackRestBackup{RepoName: "testRepo"},
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: existingTrigger
spec:
  backups:
    pgbackrest:
      manual:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      manual:
        repoName: testRepo
			`),
		},
		{
			Name:   "NewOptionsButRepo",
			Backup: pgBackRestBackup{Options: []string{"a", "b c"}},
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: existingTrigger
spec:
  backups:
    pgbackrest:
      manual:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-backup: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      manual:
        options:
        - a
        - b c
			`),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var intent unstructured.Unstructured
			assert.NilError(t, yaml.Unmarshal([]byte(tt.Before), &intent.Object))

			assert.NilError(t, tt.Backup.modifyIntent(&intent, now))
			assert.Assert(t, cmp.MarshalMatches(&intent, tt.After))
		})
	}

	t.Run("UnexpectedStructure", func(t *testing.T) {
		var intent unstructured.Unstructured
		assert.NilError(t, yaml.Unmarshal(
			[]byte(`{ spec: { backups: 1234 } }`), &intent.Object,
		))

		err := pgBackRestBackup{Options: []string{"a"}}.modifyIntent(&intent, now)
		assert.ErrorContains(t, err, ".spec.backups")
		assert.ErrorContains(t, err, "is not a map")

		err = pgBackRestBackup{RepoName: "b"}.modifyIntent(&intent, now)
		assert.ErrorContains(t, err, ".spec.backups")
		assert.ErrorContains(t, err, "is not a map")
	})
}
