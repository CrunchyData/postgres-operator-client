// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Executor calls commands
type Executor func(
	stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// pgBackRestInfo defines a pgBackRest info command with relevant flags set
func (exec Executor) pgBackRestInfo(output, repoNum string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	var command string

	command = "pgbackrest info --output=" + output
	if repoNum != "" {
		command += " --repo=" + repoNum
	}
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// bashCommand defines a one-line bash command to exec in a container
func (exec Executor) bashCommand(command string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)
	return stdout.String(), stderr.String(), err
}

// pgBackRestCheck defines a pgBackRest check command
// Force log-level-console=detail to override if set elsewhere
func (exec Executor) pgBackRestCheck() (string, string, error) {
	var stdout, stderr bytes.Buffer
	command := "pgbackrest check --log-level-console=detail"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// postgresqlListLogFiles returns the full path of numLogs log files.
func (exec Executor) listPGLogFiles(numLogs int, hasInstrumentation bool) (string, string, error) {
	var stdout, stderr bytes.Buffer

	location := "pgdata/pg[0-9][0-9]/log/*"
	if hasInstrumentation {
		location = "pgdata/logs/postgres/*.*"
	}
	// Check both the older and the newer log locations.
	// If a cluster has used both locations, this will return logs from both.
	// If a cluster does not have one location or the other, continue without error.
	// This ensures we get as many logs as possible for whatever combination of
	// log files exists on this cluster.
	// Note the "*.*" pattern to exclude the `receiver` directory,
	// which only the collector container has permission to access.
	command := fmt.Sprintf("ls -1dt %s | head -%d",
		location, numLogs)
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// listPGConfFiles returns the full path of Postgres conf files.
// These are the *.conf stored on the Postgres instance
func (exec Executor) listPGConfFiles() (string, string, error) {
	var stdout, stderr bytes.Buffer

	command := "ls -1dt pgdata/pg[0-9][0-9]/*.conf"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// listBackrestLogFiles returns the full path of pgBackRest log files.
// These are the pgBackRest logs stored on the Postgres instance
func (exec Executor) listBackrestLogFiles() (string, string, error) {
	var stdout, stderr bytes.Buffer

	// Note the "*.*" pattern to exclude the `receiver` directory,
	// which only the collector container has permission to access.
	command := "ls -1dt pgdata/pgbackrest/log/*.*"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// listPatroniLogFiles returns the full path of Patroni log file.
// These are the Patroni logs stored on the Postgres instance.
func (exec Executor) listPatroniLogFiles() (string, string, error) {
	var stdout, stderr bytes.Buffer

	// Note the "*.*" pattern to exclude the `receiver` directory,
	// which only the collector container has permission to access.
	command := "ls -1dt pgdata/patroni/log/*.*"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// listBackrestRepoHostLogFiles returns the full path of pgBackRest log files.
// These are the pgBackRest logs stored on the repo host
func (exec Executor) listBackrestRepoHostLogFiles() (string, string, error) {
	var stdout, stderr bytes.Buffer

	// Note the "*.*" pattern to exclude the `receiver` directory,
	// which only the collector container has permission to access.
	command := "ls -1dt pgbackrest/*/log/*.*"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// catFile takes the full path of a file and returns the contents
// of that file
func (exec Executor) catFile(filePath string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	command := fmt.Sprintf("cat %s", filePath)
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// copyFile takes the full path of a file and a local destination to save the
// file on disk
func (exec Executor) copyFile(source string, destination *os.File) (string, error) {
	var stderr bytes.Buffer
	command := fmt.Sprintf("cat %s", source)
	err := exec(nil, destination, &stderr, "bash", "-ceu", "--", command)
	return stderr.String(), err
}

// patronictl takes a patronictl subcommand and returns the output of that command
func (exec Executor) patronictl(cmd, output string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	command := "patronictl " + cmd
	if output != "" {
		command += " --format " + output
	}
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// processes returns the output of a ps command
func (exec Executor) processes() (string, string, error) {
	var stdout, stderr bytes.Buffer

	command := "ps aux --width 500"
	err := exec(nil, &stdout, &stderr, "bash", "-ceu", "--", command)

	return stdout.String(), stderr.String(), err
}

// systemTime returns the output of the date command
func (exec Executor) systemTime() (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := exec(nil, &stdout, &stderr, "date")
	return stdout.String(), stderr.String(), err
}
