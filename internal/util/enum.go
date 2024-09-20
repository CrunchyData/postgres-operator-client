// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import "errors"

// Define custom value types to use as flags for certain commands.
// Cobra uses the pflag package and allows custom value types by implementing
// the pflag.Value interface.
// - https://github.com/spf13/pflag
// - https://pkg.go.dev/github.com/spf13/pflag#Value

// 'patroni list' output format options
// - https://patroni.readthedocs.io/en/latest/patronictl.html#patronictl-list
// Note: Patroni has been updated to restrict the input of `--format`,
// so we can remove this when our lowest supported version of Patroni has this fix.
// - https://github.com/zalando/patroni/commit/8adddb3467f3c43ddf4ff723a2381e0cf6e2a31b
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
	return "string"
}

// 'pgbackrest info' output format options
// - https://pgbackrest.org/command.html#command-info
// `pgbackrest info` does return an error if the output is not an accepted format
// but without this enum, that error is unclear:
// Without this enum code: `Error: command terminated with exit code 32`
// With this enum code: `Error: invalid argument "jsob" for "-o, --output" flag: must be one of "text", "json"`
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
	return "string"
}
