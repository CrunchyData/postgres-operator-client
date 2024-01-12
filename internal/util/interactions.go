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
		fmt.Fprint(writer, "Please type yes or no and then press enter: ")
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
		fmt.Fprint(writer, "Please type yes or no and then press enter: ")
		return nil
	}
}
