// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestFileSizeReport(t *testing.T) {
	testsCases := []struct {
		desc   string
		bytes  float64
		output string
	}{
		{"Zero value", 0, preBox + fmt.Sprintf(msg1, 0/mebibyte) + postBox + "\n"},
		{"Less than 25 MiB", 10000, preBox + fmt.Sprintf(msg1, 10000/mebibyte) + postBox + "\n"},
		{"25 MiB", 26214400, preBox + fmt.Sprintf(msg1, 26214400/mebibyte) + postBox + "\n"},
		{"25 MiB + 1 byte", 26214401, preBox + fmt.Sprintf(msg2, 26214400/mebibyte) + postBox + "\n"},
		{"3 GiB", 3221225472, preBox + fmt.Sprintf(msg2, 3221225472/mebibyte) + postBox + "\n"},
		{"Something went wrong...", -1, preBox + fmt.Sprintf(msg1, -1/mebibyte) + postBox + "\n"},
	}

	for _, tc := range testsCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, exportSizeReport(tc.bytes), tc.output)
		})
	}
}
