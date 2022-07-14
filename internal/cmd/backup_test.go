// Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestGenerateBackupPatch(t *testing.T) {
	timestamp := time.Now().Format(time.Stamp)

	t.Run("flags present, single options", func(t *testing.T) {
		output, err := generateBackupPatch(timestamp,
			"testRepo",
			[]string{"--quoth=raven --midnight=dreary"})

		assert.NilError(t, err)

		stringOutput := string(output)
		assert.Equal(t, stringOutput, fmt.Sprintf(`{"metadata":{"annotations":`+
			`{"postgres-operator.crunchydata.com/pgbackrest-backup":"%s"}},`+
			`"spec":{"backups":{"pgbackrest":{"manual":{"options":`+
			`["--quoth=raven --midnight=dreary"],"repoName":"testRepo"}}}}}`, timestamp))
	})

	t.Run("flags present, multiple options", func(t *testing.T) {
		output, err := generateBackupPatch(timestamp,
			"testRepo",
			[]string{"--quoth=raven --midnight=dreary", "--ever=never"})

		assert.NilError(t, err)

		stringOutput := string(output)
		assert.Equal(t, stringOutput, fmt.Sprintf(`{"metadata":{"annotations":`+
			`{"postgres-operator.crunchydata.com/pgbackrest-backup":"%s"}},`+
			`"spec":{"backups":{"pgbackrest":{"manual":{"options":`+
			`["--quoth=raven --midnight=dreary","--ever=never"],"repoName":"testRepo"}}}}}`,
			timestamp))
	})

	t.Run("flags absent", func(t *testing.T) {
		output, err := generateBackupPatch(timestamp,
			"",
			[]string{})

		assert.NilError(t, err)

		stringOutput := string(output)
		assert.Equal(t, stringOutput, fmt.Sprintf(`{"metadata":{"annotations":`+
			`{"postgres-operator.crunchydata.com/pgbackrest-backup":"%s"}}}`, timestamp))
	})
}
