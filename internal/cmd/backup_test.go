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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
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

func TestBackupRun(t *testing.T) {
	cf := genericclioptions.NewConfigFlags(true)
	nsd := "test"
	cf.Namespace = &nsd
	config := &internal.Config{
		ConfigFlags: cf,
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr},
		Patch: internal.PatchConfig{FieldManager: filepath.Base(os.Args[0])},
	}

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	// Set up dynamicResourceClient with `fake` client
	gvk := v1beta1.GroupVersion.WithKind("PostgresCluster")
	gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: "postgresclusters"}
	drc := client.Resource(gvr)
	cmd := &cobra.Command{}

	t.Run("PassesThroughError", func(t *testing.T) {
		// Have the client return an error on get
		client.PrependReactor("get",
			"postgresclusters",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("whoops")
			})

		backup := pgBackRestBackup{
			Config: config,
		}

		err := backup.Run(drc, cmd, "name")
		assert.Error(t, err, "whoops", "Error from PGO API should be passed through")
	})

}
