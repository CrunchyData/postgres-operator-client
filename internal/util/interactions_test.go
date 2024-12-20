// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestConfirm(t *testing.T) {

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
			confirmed := Confirm(reader, &writer)
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
