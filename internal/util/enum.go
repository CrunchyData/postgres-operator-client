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

package util

import "errors"

// 'patroni list' output format options
// - https://patroni.readthedocs.io/en/latest/patronictl.html#patronictl-list
type patroniFormat string

const (
	PrettyPatroni patroniFormat = "pretty"
	TSVPatroni    patroniFormat = "tsv"
	JSONPatroni   patroniFormat = "json"
	YAMLPatroni   patroniFormat = "yaml"
)

// String is used both by fmt.Print and by Cobra in help text
func (e *patroniFormat) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *patroniFormat) Set(v string) error {
	switch v {
	case "pretty", "tsv", "json", "yaml":
		*e = patroniFormat(v)
		return nil
	default:
		return errors.New(`must be one of "pretty", "tsv", "json", "yaml"`)
	}
}

// Type is only used in help text
func (e *patroniFormat) Type() string {
	return "patroniFormat"
}

// 'pgbackrest info' output format options
// - https://pgbackrest.org/command.html#command-info
type pgbackrestFormat string

const (
	TextPGBackRest pgbackrestFormat = "text"
	JSONPGBackRest pgbackrestFormat = "json"
)

// String is used both by fmt.Print and by Cobra in help text
func (e *pgbackrestFormat) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *pgbackrestFormat) Set(v string) error {
	switch v {
	case "text", "json":
		*e = pgbackrestFormat(v)
		return nil
	default:
		return errors.New(`must be one of "text", "json"`)
	}
}

// Type is only used in help text
func (e *pgbackrestFormat) Type() string {
	return "pgbackrestFormat"
}
