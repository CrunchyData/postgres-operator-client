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
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/dynamic/fake"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
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

func TestCreate(t *testing.T) {
	streams, inStream, outStream, errStream := genericiooptions.NewTestIOStreams()
	cf := genericclioptions.NewConfigFlags(true)
	nsd := "test"
	cf.Namespace = &nsd
	config := &internal.Config{
		ConfigFlags: cf,
		IOStreams:   streams,
		Patch:       internal.PatchConfig{FieldManager: filepath.Base(os.Args[0])},
	}
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)

	gvk := v1beta1.GroupVersion.WithKind("PostgresCluster")

	mapper, err := config.ToRESTMapper()
	assert.NilError(t, err)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	assert.NilError(t, err)
	drc := client.Resource(mapping.Resource)
	// log.Printf("MAPPING IN TEST %#v \n", mapping)
	// client.Resource(schema.GroupVersionResource{Group: "group", Version: "version", Resource: "thekinds"})

	postgresCluster := createPostgresCluster{
		Config:         config,
		Client:         drc,
		PgMajorVersion: "14",
		ClusterName:    "hippo",
	}

	err = postgresCluster.Run(context.TODO())
	assert.NilError(t, err)

	// list, err := drc.List(context.TODO(), metav1.ListOptions{})
	// assert.NilError(t, err)
	// log.Printf("list %s", list)

	get, err := drc.Namespace("test").Get(context.TODO(), "hippo", metav1.GetOptions{})
	assert.NilError(t, err)
	log.Printf("get %s", get)

	log.Printf("in %s", inStream)
	log.Printf("out %s", outStream)
	log.Printf("err %s", errStream)
	assert.Assert(t, false)
}
