// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
