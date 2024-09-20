// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bufio"
	"fmt"
	"io"

	"k8s.io/utils/strings/slices"
)

// Confirm uses a Scanner to parse user input. A user must type in "yes" or "no"
// and then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES",
// and "Yes" all count as confirmations and return 'true'. Similarly, "n", "N",
// "no", "No", "NO" all deny confirmation and return 'false'. If the input is not
// recognized, nil is returned.
func Confirm(reader io.Reader, writer io.Writer) *bool {
	var response string
	var boolVar bool

	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		response = scanner.Text()
	}

	if scanner.Err() != nil || response == "" {
		_, _ = fmt.Fprint(writer, "Please type yes or no and then press enter: ")
		return nil
	}

	yesResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	noResponses := []string{"n", "N", "no", "No", "NO"}
	if slices.Contains(yesResponses, response) {
		boolVar = true
		return &boolVar
	} else if slices.Contains(noResponses, response) {
		return &boolVar
	} else {
		_, _ = fmt.Fprint(writer, "Please type yes or no and then press enter: ")
		return nil
	}
}
