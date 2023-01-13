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
		_, _, err := Executor(exec).patronictl("sub-command")
		assert.ErrorContains(t, err, "pass-through")

	})

}
