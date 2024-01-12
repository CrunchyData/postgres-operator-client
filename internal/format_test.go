// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
)

func TestFormatExample(t *testing.T) {
	t.Run("BlankLines", func(t *testing.T) {
		// Tabs should be replaced by spaces
		formatted := FormatExample(`
    # spaced
    a --b c

	# tabbed
	x y z`)

		expected := `
    # spaced
    a --b c

    # tabbed
    x y z`
		assert.Equal(t, expected, formatted)
	})
}
