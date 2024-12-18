// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"io"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPGBackRestInfo(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "pgbackrest info --output=text"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).pgBackRestInfo("text", "")
		assert.ErrorContains(t, err, "pass-through")

	})

	t.Run("text, repo 1", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "pgbackrest info --output=text --repo=1"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).pgBackRestInfo("text", "1")
		assert.ErrorContains(t, err, "pass-through")

	})

	t.Run("json, repo 2", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "pgbackrest info --output=json --repo=2"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).pgBackRestInfo("json", "2")
		assert.ErrorContains(t, err, "pass-through")

	})
}

func TestListPGLogFiles(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "ls -1dt pgdata/pg[0-9][0-9]/log/* | head -1"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).listPGLogFiles(1)
		assert.ErrorContains(t, err, "pass-through")

	})

}

func TestListPatroniLogFiles(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "ls -1dt pgdata/patroni/log/*"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).listPatroniLogFiles()
		assert.ErrorContains(t, err, "pass-through")

	})

}

func TestCatFile(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "cat /path/to/file"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).catFile("/path/to/file")
		assert.ErrorContains(t, err, "pass-through")

	})

}

func TestPatronictl(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "patronictl sub-command"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).patronictl("sub-command", "")
		assert.ErrorContains(t, err, "pass-through")

	})

}

func TestProcesses(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.DeepEqual(t, command, []string{"bash", "-ceu", "--", "ps aux --width 500"})
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")
			return expected
		}
		_, _, err := Executor(exec).processes()
		assert.ErrorContains(t, err, "pass-through")

	})

}
