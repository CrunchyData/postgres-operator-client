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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

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

func TestCreateArgsErrors(t *testing.T) {
	streams, _, _, _ := genericiooptions.NewTestIOStreams()
	config := &internal.Config{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		Patch:       internal.PatchConfig{FieldManager: filepath.Base(os.Args[0])},
	}

	cmd := newCreateClusterCommand(config)
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)

	for _, test := range []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name:     "missing arg",
			args:     []string{},
			errorMsg: "accepts 1 arg(s), received 0",
		},
		{
			name:     "too many args",
			args:     []string{"hippo", "rhino"},
			errorMsg: "accepts 1 arg(s), received 2",
		},
		{
			name:     "missing version flag arg",
			args:     []string{"hippo"},
			errorMsg: "\"pg-major-version\" not set",
		},
		{
			name:     "flag present but unset",
			args:     []string{"hippo", "--pg-major-version="},
			errorMsg: "invalid argument \"\" for \"--pg-major-version\" flag: strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			name:     "wrong type for version",
			args:     []string{"hippo", fmt.Sprintf("--pg-major-version=%f", 15.3)},
			errorMsg: "invalid argument \"15.300000\" for \"--pg-major-version\" flag: strconv.ParseInt: parsing \"15.300000\": invalid syntax",
		},
		{
			name:     "wrong type for version",
			args:     []string{"hippo", fmt.Sprintf("--pg-major-version=%s", "x")},
			errorMsg: "invalid argument \"x\" for \"--pg-major-version\" flag: strconv.ParseInt: parsing \"x\": invalid syntax",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd.SetArgs(test.args)
			cmd.Execute()
			assert.Assert(t, strings.Contains(buf.String(), test.errorMsg),
				fmt.Sprintf("Expected '%s', got '%s'\n", test.errorMsg, buf.String()))
			// Clear out buffer
			buf.Reset()
		})
	}
}

func TestCreatePassesThroughError(t *testing.T) {

	streams, _, _, _ := genericiooptions.NewTestIOStreams()
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
	// Have the client return an error on creates
	client.PrependReactor("create",
		"postgresclusters",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("whoops")
		})

	// Set up dynamicResourceClient with `fake` client
	gvk := v1beta1.GroupVersion.WithKind("PostgresCluster")
	drc := client.Resource(schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: "postgresclusters"})

	postgresCluster := createPostgresCluster{
		Config:         config,
		Client:         drc,
		PgMajorVersion: 14,
		ClusterName:    "hippo",
	}

	err := postgresCluster.Run(context.TODO())
	assert.Error(t, err, "whoops", "Error from PGO API should be passed through")
}

// func TestCreate(t *testing.T) {

// 	streams, inStream, outStream, errStream := genericiooptions.NewTestIOStreams()
// 	cf := genericclioptions.NewConfigFlags(true)
// 	nsd := "default"
// 	cf.Namespace = &nsd
// 	config := &internal.Config{
// 		ConfigFlags: cf,
// 		IOStreams:   streams,
// 		Patch:       internal.PatchConfig{FieldManager: filepath.Base(os.Args[0])},
// 	}

// 	cmd := newCreateClusterCommand(config)
// 	buf := new(bytes.Buffer)
// 	cmd.SetOutput(buf)
// 	cmd.SetArgs([]string{
// 		"hippo2",
// 		fmt.Sprintf("--pg-major-version=%f", 15.1),
// 	})
// 	cmd.Execute()
// 	log.Printf("in %s", inStream)
// 	log.Printf("out %s", outStream)
// 	log.Printf("err %s", errStream)
// 	t.Logf("hey %s", buf)

// 	scheme := runtime.NewScheme()
// 	client := fake.NewSimpleDynamicClient(scheme)

// 	client.PrependReactor("create", "postgresclusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
// 		t.Logf("This works :)")
// 		return true, nil, nil
// 		// fmt.Errorf("whoops")
// 	})

// 	gvk := v1beta1.GroupVersion.WithKind("PostgresCluster")

// 	mapper, err := config.ToRESTMapper()
// 	assert.NilError(t, err)

// 	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
// 	assert.NilError(t, err)
// 	drc := client.Resource(mapping.Resource)
// 	// log.Printf("MAPPING IN TEST %#v \n", mapping)
// 	// client.Resource(schema.GroupVersionResource{Group: "group", Version: "version", Resource: "thekinds"})

// 	postgresCluster := createPostgresCluster{
// 		Config:         config,
// 		Client:         drc,
// 		PgMajorVersion: 14,
// 		ClusterName:    "hippo",
// 	}

// 	err = postgresCluster.Run(context.TODO())
// 	assert.NilError(t, err)

// 	// list, err := drc.List(context.TODO(), metav1.ListOptions{})
// 	// assert.NilError(t, err)
// 	// log.Printf("list %s", list)

// 	get, err := drc.Namespace("test").Get(context.TODO(), "hippo", metav1.GetOptions{})
// 	assert.NilError(t, err)
// 	log.Printf("get %s", get)

// 	log.Printf("in %s", inStream)
// 	log.Printf("out %s", outStream)
// 	log.Printf("err %s", errStream)
// 	assert.Assert(t, false)
// }
