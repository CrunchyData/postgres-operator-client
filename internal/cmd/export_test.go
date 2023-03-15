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
		{"Zero value", 0, fmt.Sprintf(msg1, 0/mebibyte)},
		{"Less than 25 MiB", 10000, fmt.Sprintf(msg1, 10000/mebibyte)},
		{"25 MiB", 26214400, fmt.Sprintf(msg1, 26214400/mebibyte)},
		{"25 MiB + 1 byte", 26214401,
			fmt.Sprintf(msg1, 26214401/mebibyte) + fmt.Sprintf(msg2, 26214400/mebibyte)},
		{"3 GiB", 3221225472,
			fmt.Sprintf(msg1, 3221225472/mebibyte) + fmt.Sprintf(msg2, 3221225472/mebibyte)},
		{"Something went wrong...", -1, fmt.Sprintf(msg1, -1/mebibyte)},
	}

	for _, tc := range testsCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, exportSizeReport(tc.bytes), tc.output)
		})
	}
}
