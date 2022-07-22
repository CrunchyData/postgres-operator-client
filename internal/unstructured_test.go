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
	"strings"
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

func TestMergeStringMaps(t *testing.T) {
	assert.DeepEqual(t, MergeStringMaps(), map[string]string{})

	assert.DeepEqual(t, MergeStringMaps(
		map[string]string{"a": "1", "b": "2"},
		map[string]string{"m": "x", "n": "y"},
		map[string]string{"c": "6", "d": "9"},
	), map[string]string{
		"a": "1", "b": "2", "c": "6", "d": "9", "m": "x", "n": "y",
	})

	assert.DeepEqual(t, MergeStringMaps(
		map[string]string{"a": "1", "b": "2"},
		map[string]string{"a": "3", "c": "4"},
		map[string]string{"b": "5", "d": "6"},
	), map[string]string{
		"a": "3", "b": "5", "c": "4", "d": "6",
	})
}

func TestRemoveEmptyField(t *testing.T) {
	var object unstructured.Unstructured
	assert.NilError(t, yaml.Unmarshal([]byte(strings.TrimSpace(`
string:
  zero: ""
  full: asdf

integer:
  zero: 0
  full: 99

boolean:
  zero: false
  full: true

array:
  empty: []
  full: [asdf]

object:
  empty: {}
  full:
    some: true

blank:
	`)), &object.Object))

	t.Run("String", func(t *testing.T) {
		RemoveEmptyField(&object, "string", "full")
		assert.Assert(t, cmp.MarshalMatches(object.Object["string"], `
full: asdf
zero: ""
		`))

		RemoveEmptyField(&object, "string", "zero")
		assert.Assert(t, cmp.MarshalMatches(object.Object["string"], `
full: asdf
		`))
	})

	t.Run("Integer", func(t *testing.T) {
		RemoveEmptyField(&object, "integer", "full")
		assert.Assert(t, cmp.MarshalMatches(object.Object["integer"], `
full: 99
zero: 0
		`))

		RemoveEmptyField(&object, "integer", "zero")
		assert.Assert(t, cmp.MarshalMatches(object.Object["integer"], `
full: 99
		`))
	})

	t.Run("Boolean", func(t *testing.T) {
		RemoveEmptyField(&object, "boolean", "full")
		assert.Assert(t, cmp.MarshalMatches(object.Object["boolean"], `
full: true
zero: false
		`))

		RemoveEmptyField(&object, "boolean", "zero")
		assert.Assert(t, cmp.MarshalMatches(object.Object["boolean"], `
full: true
		`))
	})

	t.Run("Array", func(t *testing.T) {
		RemoveEmptyField(&object, "array", "full")
		assert.Assert(t, cmp.MarshalMatches(object.Object["array"], `
empty: []
full:
- asdf
		`))

		RemoveEmptyField(&object, "array", "empty")
		assert.Assert(t, cmp.MarshalMatches(object.Object["array"], `
full:
- asdf
		`))
	})

	t.Run("Object", func(t *testing.T) {
		RemoveEmptyField(&object, "object", "full")
		assert.Assert(t, cmp.MarshalMatches(object.Object["object"], `
empty: {}
full:
  some: true
		`))

		RemoveEmptyField(&object, "object", "empty")
		assert.Assert(t, cmp.MarshalMatches(object.Object["object"], `
full:
  some: true
		`))
	})

	t.Run("Blank", func(t *testing.T) {
		_, exists, _ := unstructured.NestedFieldNoCopy(object.Object, "blank")
		assert.Assert(t, exists)

		RemoveEmptyField(&object, "blank")

		_, exists, _ = unstructured.NestedFieldNoCopy(object.Object, "blank")
		assert.Assert(t, !exists)
	})
}

func TestRemoveEmptySections(t *testing.T) {
	var object unstructured.Unstructured
	assert.NilError(t, yaml.Unmarshal([]byte(strings.TrimSpace(`
spec:
  section:
    other:
    empty:
      more:
        than:
          once: {}
	`)), &object.Object))

	RemoveEmptySections(&object, "spec", "section", "empty", "more", "than", "once")

	assert.Assert(t, cmp.MarshalMatches(&object, `
spec:
  section:
    other: null
	`))
}
