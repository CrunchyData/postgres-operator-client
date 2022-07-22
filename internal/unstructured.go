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
	"bytes"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

// ExtractFieldsInto copies the fields owned by fieldManager from the main
// resource (not a subresource) of src into dst.
func ExtractFieldsInto(src, dst *unstructured.Unstructured, fieldManager string) error {
	entry, ok := findManagedFields(src, fieldManager, "")
	if !ok || entry.FieldsV1 == nil {
		dst.SetGroupVersionKind(src.GroupVersionKind())
		return nil
	}

	var paths fieldpath.Set
	if err := paths.FromJSON(bytes.NewReader(entry.FieldsV1.Raw)); err != nil {
		return fmt.Errorf("cannot unmarshal FieldsV1 from JSON: %w", err)
	}

	result, err := extractDepthFirst(src.Object, paths.Leaves())
	if err == nil {
		dst.Object, _ = result.(map[string]interface{})
		dst.SetGroupVersionKind(src.GroupVersionKind())
	}

	return err
}

// extractDepthFirst creates a copy of src consisting exclusively of fields
// defined in paths. It ignores fields that do not exist in src or otherwise
// cannot be reached.
func extractDepthFirst(src interface{}, paths *fieldpath.Set) (interface{}, error) {
	// The iterators below do not return errors. Every closure must assign this
	// variable and exit early when it is set.
	var err error
	var result interface{}

	// Each child represents an intermediate (non-leaf) step in a field path.
	// Visiting these before paths.Members makes this recursion depth-first.
	paths.Children.Iterate(func(pe fieldpath.PathElement) {
		switch {

		// Something already failed; do nothing.
		case err != nil:

		// The containers are objects. Descend into the src field value and
		// assign to the result.
		case pe.FieldName != nil:
			srcMap, _ := src.(map[string]interface{})
			srcField, ok, _ := unstructured.NestedFieldNoCopy(srcMap, *pe.FieldName)

			if ok && result == nil {
				result = make(map[string]interface{})
			}

			resultMap, _ := result.(map[string]interface{})
			if ok && resultMap != nil {
				if nextSet, ok := paths.Children.Get(pe); ok {
					resultMap[*pe.FieldName], err = extractDepthFirst(srcField, nextSet)
				}
			}

		// The containers are lists of objects identified by a handful of fields,
		// that is, "+listType=map" or "x-kubernetes-list-type: map". Find the
		// identified object, descend into it, assign its identity fields, and
		// append it to the result.
		// - https://docs.k8s.io/reference/using-api/server-side-apply/#merge-strategy
		case pe.Key != nil:
			srcSlice, _ := src.([]interface{})
			for i := range srcSlice {
				srcField, _ := srcSlice[i].(map[string]interface{})
				if err != nil || !objectHasKey(srcField, *pe.Key) {
					continue
				}

				if result == nil {
					result = make([]interface{}, 0, 1)
				}

				resultSlice, _ := result.([]interface{})
				if resultSlice != nil {
					var resultValue interface{}

					if nextSet, ok := paths.Children.Get(pe); ok {
						resultValue, err = extractDepthFirst(srcField, nextSet)
					}

					resultValueMap, _ := resultValue.(map[string]interface{})
					if resultValueMap != nil {
						for _, keyField := range *pe.Key {
							resultValueMap[keyField.Name] = keyField.Value.Unstructured()
						}
					}

					result = append(resultSlice, resultValue)
				}
			}

		default:
			err = fmt.Errorf("unexpected PathElement: %q", pe)
		}
	})

	// Each member represents the last element (leaf) of a field path.
	paths.Members.Iterate(func(pe fieldpath.PathElement) {
		switch {

		// Something already failed; do nothing.
		case err != nil:

		// The containers are objects. Copy the field value from src to the result.
		case pe.FieldName != nil:
			srcMap, _ := src.(map[string]interface{})
			srcField, ok, _ := unstructured.NestedFieldCopy(srcMap, *pe.FieldName)

			if ok && result == nil {
				result = make(map[string]interface{})
			}

			resultMap, _ := result.(map[string]interface{})
			if ok && resultMap != nil {
				resultMap[*pe.FieldName] = srcField
			}

		// The containers are lists of scalar values, that is, "+listType=set"
		// or "x-kubernetes-list-type: set". Append the managed value to the result.
		// - https://docs.k8s.io/reference/using-api/server-side-apply/#merge-strategy
		case pe.Value != nil:
			if result == nil {
				result = make([]interface{}, 0, 1)
			}

			resultSlice, _ := result.([]interface{})
			if resultSlice != nil {
				result = append(resultSlice, (*pe.Value).Unstructured())
			}

		default:
			err = fmt.Errorf("unexpected PathElement: %q", pe)
		}
	})

	return result, err
}

// findManagedFields returns the server-side apply entry on object for
// fieldManager and subresource. Blank subresource represents the main resource.
func findManagedFields(object metav1.Object, fieldManager, subresource string) (metav1.ManagedFieldsEntry, bool) {
	for _, mfe := range object.GetManagedFields() {
		if mfe.Manager == fieldManager && mfe.Subresource == subresource &&
			mfe.Operation == metav1.ManagedFieldsOperationApply {
			return mfe, true
		}
	}

	return metav1.ManagedFieldsEntry{}, false
}

// objectHasKey returns whether or not object has all the fields and values in key.
func objectHasKey(object map[string]interface{}, key value.FieldList) bool {
	for i := range key {
		v, ok := object[key[i].Name]
		if !ok || !value.Equals(key[i].Value, value.NewValueInterface(v)) {
			return false
		}
	}
	return true
}

// MergeStringMaps returns a new map that contains all the keys and values in
// maps. When two or more maps have the same key, the value from the rightmost
// map is used.
func MergeStringMaps(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

// RemoveEmptyField removes a nested field from object when it is an empty map
// or slice or it is the zero value for its type.
func RemoveEmptyField(object *unstructured.Unstructured, fields ...string) {
	value, _, _ := unstructured.NestedFieldNoCopy(object.Object, fields...)
	rv := reflect.ValueOf(value)

	if value == nil || rv.IsZero() {
		unstructured.RemoveNestedField(object.Object, fields...)
		return
	}

	if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map) && rv.Len() == 0 {
		unstructured.RemoveNestedField(object.Object, fields...)
		return
	}
}

// RemoveEmptySections removes an empty nested field from object the same as
// [RemoveEmptyField] then removes all its empty parent fields.
func RemoveEmptySections(object *unstructured.Unstructured, sections ...string) {
	for i := len(sections); i > 0; i-- {
		RemoveEmptyField(object, sections[:i]...)
	}
}
