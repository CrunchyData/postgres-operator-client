package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/crunchydata/postgres-operator-client/internal/cmd"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
---
`

func main() {

	fmt.Println("generate CLI markdown")

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		fmt.Println(base)
		return fmt.Sprintf(fmTemplate, strings.ReplaceAll(base, "_", " "))
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/reference/" + strings.ToLower(base) + "/"
	}

	pgo := cmd.NewPGOCommand(os.Stdin, os.Stdout, os.Stderr)
	err := doc.GenMarkdownTreeCustom(pgo, "./", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
