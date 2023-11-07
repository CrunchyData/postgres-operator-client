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
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator-client/internal/util"
)

func TestConfirmDelete(t *testing.T) {

	testsCases := []struct {
		input           string
		invalidResponse bool
		confirmed       bool
	}{
		{"abc", true, false}, // invalid
		{"", true, false},    // invalid
		{"y", false, true},
		{"Y", false, true},
		{"yes", false, true},
		{"Yes", false, true},
		{"YES", false, true},
		{"n", false, false},
		{"N", false, false},
		{"no", false, false},
		{"No", false, false},
		{"NO", false, false},
		{"yep", true, false},  // invalid
		{"nope", true, false}, // invalid
	}

	for _, tc := range testsCases {
		t.Run("input is "+tc.input, func(t *testing.T) {
			var reader io.Reader = strings.NewReader(tc.input)
			var writer bytes.Buffer
			confirmed := util.Confirm(reader, &writer)
			if tc.invalidResponse {
				assert.Assert(t, confirmed == nil)
				response, err := writer.ReadString(':')
				assert.NilError(t, err)
				assert.Equal(t, response, "Please type yes or no and then press enter:")

			} else {
				assert.Assert(t, confirmed != nil)
				assert.Assert(t, *confirmed == tc.confirmed)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	testsCases := []struct {
		desc    string
		slice   []string
		element string
		found   bool
	}{
		{"found", []string{"a", "b", "c"}, "a", true},
		{"not found", []string{"a", "b", "c"}, "x", false},
		{"not found substring", []string{"ab", "bc", "cd"}, "b", false},
	}

	for _, tc := range testsCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, containsString(tc.slice, tc.element), tc.found)
		})
	}
}
