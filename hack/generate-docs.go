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

//go:build docs

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator-client/internal/cmd"
)

func main() {
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, filepath.Ext(name))
		command := strings.ReplaceAll(base, "_", " ")

		// https://gohugo.io/content-management/front-matter/
		front, _ := yaml.Marshal(map[string]any{
			"title": command,
		})
		return "---\n" + string(front) + "---\n"
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if base == "pgo" {
			return "/reference/"
		}
		return "/reference/" + strings.ToLower(base) + "/"
	}

	pgo := cmd.NewPGOCommand(os.Stdin, os.Stdout, os.Stderr)
	err := doc.GenMarkdownTreeCustom(pgo, "./", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
