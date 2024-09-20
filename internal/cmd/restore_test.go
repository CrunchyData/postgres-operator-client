// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

func TestPGBackRestRestoreModifyIntent(t *testing.T) {
	now := time.Date(2020, 4, 5, 6, 7, 8, 99, time.FixedZone("ZONE", -11))

	for _, tt := range []struct {
		Name, Before, After string
		Restore             pgBackRestRestore
	}{
		{
			Name: "Zero",
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
			`),
		},
		{
			Name: "Options",
			Restore: pgBackRestRestore{
				Options: []string{"--quoth=raven --midnight=dreary", "--ever=never"},
			},
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        options:
        - --quoth=raven --midnight=dreary
        - --ever=never
			`),
		},
		{
			Name:    "RepoName",
			Restore: pgBackRestRestore{RepoName: "testRepo"},
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        repoName: testRepo
			`),
		},
		{
			Name: "OldRepoAndOptions",
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: existingTrigger
spec:
  backups:
    pgbackrest:
      restore:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
			`),
		},
		{
			Name:    "NewRepoButOptions",
			Restore: pgBackRestRestore{RepoName: "testRepo"},
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: existingTrigger
spec:
  backups:
    pgbackrest:
      restore:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        repoName: testRepo
			`),
		},
		{
			Name:    "NewOptionsButRepo",
			Restore: pgBackRestRestore{Options: []string{"a", "b c"}},
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: existingTrigger
spec:
  backups:
    pgbackrest:
      restore:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: "2020-04-05T06:07:19Z"
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        options:
        - a
        - b c
			`),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var intent unstructured.Unstructured
			assert.NilError(t, yaml.Unmarshal([]byte(tt.Before), &intent.Object))

			assert.NilError(t, tt.Restore.modifyIntent(&intent, now))
			assert.Assert(t, cmp.MarshalMatches(&intent, tt.After))
		})
	}

	t.Run("UnexpectedStructure", func(t *testing.T) {
		var intent unstructured.Unstructured
		assert.NilError(t, yaml.Unmarshal(
			[]byte(`{ spec: { backups: 1234 } }`), &intent.Object,
		))

		err := pgBackRestBackupArgs{Options: []string{"a"}}.modifyIntent(&intent, now)
		assert.ErrorContains(t, err, ".spec.backups")
		assert.ErrorContains(t, err, "is not a map")

		err = pgBackRestBackupArgs{RepoName: "b"}.modifyIntent(&intent, now)
		assert.ErrorContains(t, err, ".spec.backups")
		assert.ErrorContains(t, err, "is not a map")
	})
}

func TestPGBackRestRestoreDisableModifyIntent(t *testing.T) {
	for _, tt := range []struct {
		Name, Before, After string
		Disable             pgBackRestRestoreDisable
	}{
		{
			Name:  "Zero",
			After: `null`,
		},
		{
			Name: "OldRepoAndOptions",
			Before: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: existingTrigger
spec:
  backups:
    pgbackrest:
      restore:
        options: ["--from=before"]
        repoName: priorRepo
			`),
			After: strings.TrimSpace(`
metadata:
  annotations:
    postgres-operator.crunchydata.com/pgbackrest-restore: existingTrigger
			`),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var intent unstructured.Unstructured
			assert.NilError(t, yaml.Unmarshal([]byte(tt.Before), &intent.Object))

			assert.NilError(t, tt.Disable.modifyIntent(&intent))
			assert.Assert(t, cmp.MarshalMatches(&intent, tt.After))
		})
	}
}
