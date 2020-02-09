package main

import (
	"bufio"
	"bytes"
	"os"
	"strings"

	"github.com/tamasfe/repose/cmd/repose/config"
	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/common"
)

func main() {
	var parsersBuilder strings.Builder
	var transformersBuilder strings.Builder
	var generatorsBuilder strings.Builder

	for _, p := range config.Parsers {
		pMd, ok := p.(common.DescriptionMarkdown)
		if !ok {
			continue
		}

		pDesc := bufio.NewScanner(bytes.NewBufferString(pMd.DescriptionMarkdown()))
		parsersBuilder.WriteString("# " + p.Name() + "\n")

		for pDesc.Scan() {
			line := pDesc.Text()
			if len(line) != 0 && line[0] == '#' {
				line = "#" + line
			}

			parsersBuilder.WriteString(line + "\n")
		}
		parsersBuilder.WriteString("\n")
	}

	for _, t := range config.Transformers {
		tMd, ok := t.(common.DescriptionMarkdown)
		if !ok {
			continue
		}

		tDesc := bufio.NewScanner(bytes.NewBufferString(tMd.DescriptionMarkdown()))
		transformersBuilder.WriteString("# " + t.Name() + "\n")

		for tDesc.Scan() {
			line := tDesc.Text()
			if len(line) != 0 && line[0] == '#' {
				line = "#" + line
			}
			transformersBuilder.WriteString(line + "\n")
		}
		transformersBuilder.WriteString("\n")
	}

	for _, g := range config.Generators {
		genMd, ok := g.(common.DescriptionMarkdown)
		if !ok {
			continue
		}

		genDesc := bufio.NewScanner(bytes.NewBufferString(genMd.DescriptionMarkdown()))
		generatorsBuilder.WriteString("# " + g.Name() + "\n")

		for genDesc.Scan() {
			line := genDesc.Text()
			if len(line) != 0 && line[0] == '#' {
				line = "#" + line
			}
			generatorsBuilder.WriteString(line + "\n")
		}
		generatorsBuilder.WriteString("\n")
	}

	err := os.MkdirAll("./docs/cli/parsers", os.ModePerm)
	if err != nil {
		panic(err)
	}

	pf, err := os.Create("./docs/cli/parsers/README.md")
	if err != nil {
		panic(err)
	}
	defer pf.Close()
	_, err = pf.WriteString(markdown.GenTOC("# Parsers\n", parsersBuilder.String()))
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll("./docs/cli/transformers", os.ModePerm)
	if err != nil {
		panic(err)
	}

	tf, err := os.Create("./docs/cli/transformers/README.md")
	if err != nil {
		panic(err)
	}
	defer tf.Close()
	_, err = tf.WriteString(markdown.GenTOC("# Specification transformers\n", transformersBuilder.String()))
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll("./docs/cli/generators", os.ModePerm)
	if err != nil {
		panic(err)
	}

	gf, err := os.Create("./docs/cli/generators/README.md")
	if err != nil {
		panic(err)
	}
	defer gf.Close()
	_, err = gf.WriteString(markdown.GenTOC("# Code generators\n", generatorsBuilder.String()))
	if err != nil {
		panic(err)
	}
}
