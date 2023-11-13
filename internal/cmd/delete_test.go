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
)

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
