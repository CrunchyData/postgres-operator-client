// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import "strings"

// FormatExample prepares text to appear in the Examples section of a command's
// help text.
// For spacing, all tabs should be replaced with spaces
func FormatExample(text string) string {
	return strings.ReplaceAll(text, "\t", "    ")
}
