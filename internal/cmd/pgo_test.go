// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestFormatHeader(t *testing.T) {
	testsCases := []struct {
		desc   string
		input  string
		output string
	}{
		{"Expected", "### Some Header\n", "Some Header:\n"},
		{"One valid, one not", "### Some Header\n### Another Header", "Some Header:\n### Another Header"},
		{"Two expected", "### Some Header\n### Another Header\n", "Some Header:\nAnother Header:\n"},
		{"No newline", "### Some Header", "### Some Header"},
		{"2 hashes", "## Some Header\n", "## Some Header\n"},
		{"leading space", " ### Some Header\n", " Some Header:\n"},
	}

	for _, tc := range testsCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, formatHeader(tc.input), tc.output)
		})
	}
}
