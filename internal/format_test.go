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

package internal

import (
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestFormatExample(t *testing.T) {
	alwaysExpect := func(t testing.TB, formatted string) {
		assert.Assert(t, formatted[:1] != "\n", "should not start with newline")

		// Every line should be indented two spaces.
		assert.DeepEqual(t,
			strings.Split(formatted, "\n"),
			regexp.MustCompile(`(?m)^  ($|[^ ].*$)`).FindAllString(formatted, -1))
	}

	t.Run("BlankLines", func(t *testing.T) {
		formatted := FormatExample(`
# first
a --b c

# second
x y z
`)

		alwaysExpect(t, formatted)
		assert.Equal(t, "  # first\n  a --b c\n  \n  # second\n  x y z", formatted)
	})

	t.Run("TrailingTabs", func(t *testing.T) {
		formatted := FormatExample(`
# description
command with arguments
		`)

		alwaysExpect(t, formatted)
		assert.Equal(t, "  # description\n  command with arguments", formatted)
	})
}
