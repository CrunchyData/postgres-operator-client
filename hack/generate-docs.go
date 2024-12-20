// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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
