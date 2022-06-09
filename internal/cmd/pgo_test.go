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

func ExampleNewPGOCommand() {
	cmd := NewPGOCommand(nil, nil, nil)
	_ = cmd.Execute()

	// Output:
	// pgo is a kubectl plugin for PGO, the open source Postgres Operator from Crunchy Data.
	//
	//	https://github.com/CrunchyData/postgres-operator
	//
	// Usage:
	//   kubectl-pgo [command]
	//
	// Available Commands:
	//   example     short description
	//   help        Help about any command
	//
	// Flags:
	//   -h, --help   help for kubectl-pgo
	//
	// Use "kubectl-pgo [command] --help" for more information about a command.
}
