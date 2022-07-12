// Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package internal

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator-client/internal/testing/cmp"
)

func TestExtractFieldsInto(t *testing.T) {
	objectYAML := []byte(`
apiVersion: something/v1
kind: BigTime
metadata:
  annotations:
    one-piece: king
  finalizers:
  - foregroundDeletion
  - something/finalizer

  managedFields:
  - apiVersion: something/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:finalizers:
          .: {}
          v:"something/finalizer": {}
    manager: something-protector
    operation: Apply

  - apiVersion: something/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:shared:
          f:front: {}
    manager: something-protector
    operation: Update

  - apiVersion: something/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        .: {}
        f:conditions:
          .: {}
          k:{"type":"AllGood"}:
            .: {}
            f:lastTransitionTime: {}
            f:message: {}
            f:observedGeneration: {}
            f:reason: {}
            f:status: {}
            f:type: {}
    manager: something-controller
    operation: Apply
    subresource: status

  - apiVersion: something/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:listMap:
          k:{"name":"first"}:
            .: {}
            f:value: {}
        f:shared:
          f:front: {}
    manager: something-applier1
    operation: Apply

  - apiVersion: something/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:one-piece: {}
      f:spec:
        f:listMap:
          k:{"name":"last"}:
            .: {}
            f:value: {}
        f:shared:
          f:back: {}
    manager: something-applier2
    operation: Apply

spec:
  shared:
    front: panel
    back: wires
  listMap:
  - name: first
    value: roller
  - name: last
    value: skate

status:
  conditions:
  - lastTransitionTime: "2020-01-28T16:17:18Z"
    message: Things are going great
    reason: ChecksOut
    status: "True"
    type: AllGood
`)

	t.Run("NothingFromUpdater", func(t *testing.T) {
		var src, dst unstructured.Unstructured
		assert.NilError(t, yaml.Unmarshal(objectYAML, &src))

		assert.NilError(t, ExtractFieldsInto(&src, &dst, "something-controller"))
		assert.Assert(t, cmp.MarshalMatches(&dst, `
apiVersion: something/v1
kind: BigTime
		`))
	})

	t.Run("ListTypeMap", func(t *testing.T) {
		var src, dst unstructured.Unstructured
		assert.NilError(t, yaml.Unmarshal(objectYAML, &src))

		assert.NilError(t, ExtractFieldsInto(&src, &dst, "something-applier1"))
		assert.Assert(t, cmp.MarshalMatches(&dst, `
apiVersion: something/v1
kind: BigTime
spec:
  listMap:
  - name: first
    value: roller
  shared:
    front: panel
		`))

		assert.NilError(t, ExtractFieldsInto(&src, &dst, "something-applier2"))
		assert.Assert(t, cmp.MarshalMatches(&dst, `
apiVersion: something/v1
kind: BigTime
metadata:
  annotations:
    one-piece: king
spec:
  listMap:
  - name: last
    value: skate
  shared:
    back: wires
		`))
	})

	t.Run("ListTypeSet", func(t *testing.T) {
		var src, dst unstructured.Unstructured
		assert.NilError(t, yaml.Unmarshal(objectYAML, &src))

		assert.NilError(t, ExtractFieldsInto(&src, &dst, "something-protector"))
		assert.Assert(t, cmp.MarshalMatches(&dst, `
apiVersion: something/v1
kind: BigTime
metadata:
  finalizers:
  - something/finalizer
		`))
	})
}
