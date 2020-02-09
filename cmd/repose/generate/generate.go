package generate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Masterminds/sprig"
	"github.com/dave/jennifer/jen"
	"github.com/tamasfe/repose/cmd/repose/config"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/generator"
	"github.com/tamasfe/repose/pkg/parser"
	"github.com/tamasfe/repose/pkg/spec"
	"github.com/tamasfe/repose/pkg/transformer"
	"github.com/tamasfe/repose/pkg/util/cli"
)

type filenameValues struct {
	Generator string
	Target    string
}

// Generate generate code according to options
func Generate(cliOpts *config.GenerateOptions, options *config.ReposeOptions, inPaths []string) error {

	normalizeNames(options)

	// Provide all the generator options in the
	// context as well.
	ctxGeneratorOptions := make(map[string]interface{})

	for genName, gen := range options.Generators {
		ctxGeneratorOptions[genName] = gen.Options
	}

	ctx := context.WithValue(context.Background(), common.ContextGeneratorOptions, ctxGeneratorOptions)

	state := &common.State{}
	ctx = context.WithValue(ctx, common.ContextState, state)

	spec, err := parseSpec(ctx, cliOpts, options, inPaths)
	if err != nil {
		return err
	}

	transformers, err := getTransformers(options)
	if err != nil {
		return err
	}

	for i, t := range transformers {
		err = t.Transform(ctx, options.Transformers[i].Options, spec)
		if err != nil {
			return fmt.Errorf("transform failed: %w", err)
		}
	}

	return generateCode(ctx, cliOpts, options, spec)
}

func generateCode(
	ctx context.Context,
	cliOpts *config.GenerateOptions,
	options *config.ReposeOptions,
	spec *spec.Spec,
) error {
	outputToStdout := cliOpts.OutPath == "-" || cliOpts.OutPath == ""

	isOutDir := !outputToStdout && isDir(cliOpts.OutPath)

	singleFile := outputToStdout || !isOutDir

	generators, err := getGenerators(cliOpts, options)
	if err != nil {
		return err
	}

	hasGenerator := regexp.MustCompile(`\{\{\s?\.Generator\s?\}\}`)
	hasTarget := regexp.MustCompile(`\{\{\s?\.Target\s?\}\}`)

	fileNameTemplate, err := template.New("filename").Funcs(sprig.TxtFuncMap()).Parse(options.FilePattern)
	if err != nil {
		return fmt.Errorf("invalid file pattern: %w", err)
	}

	if options.PackageName == "" {
		if isOutDir {
			absPath, err := filepath.Abs(cliOpts.OutPath)
			if err != nil {
				return err
			}
			splitPath := strings.Split(absPath, string(os.PathSeparator))
			options.PackageName = splitPath[len(splitPath)-1]
			cli.Warningf("Package name is not specified, \"%v\" was inferred from the directory name.\n", options.PackageName)
		} else if !outputToStdout {
			absPath, err := filepath.Abs(cliOpts.OutPath)
			if err != nil {
				return err
			}
			splitPath := strings.Split(absPath, string(os.PathSeparator))
			options.PackageName = splitPath[len(splitPath)-2]
			cli.Warningf("Package name is not specified, \"%v\" was inferred from the directory name.\n", options.PackageName)
		} else {
			options.PackageName = "api"
			cli.Warningf("Package name is not specified, using \"%v\".\n", options.PackageName)
		}

	}

	if isOutDir {
		if len(hasGenerator.FindStringIndex(options.FilePattern)) == 0 &&
			len(hasTarget.FindStringIndex(options.FilePattern)) == 0 {

			fnBuf := &bytes.Buffer{}

			err = fileNameTemplate.Execute(fnBuf, &filenameValues{})
			if err != nil {
				return fmt.Errorf("invalid file pattern: %w", err)
			}

			if cliOpts.Yes {
				err = os.MkdirAll(cliOpts.OutPath, os.ModePerm)
				if err != nil {
					return err
				}
			}

			cliOpts.OutPath = filepath.Join(cliOpts.OutPath, fnBuf.String())
			singleFile = true
		}
	}

	if singleFile {
		allTargets := make(map[string][]string, len(generators))

		for gName, g := range options.Generators {
			allTargets[gName] = g.Targets
		}

		codeBuf := &bytes.Buffer{}

		err = generateUnit(
			ctx,
			options,
			spec,
			generators,
			allTargets,
			codeBuf,
		)

		if err != nil {
			return fmt.Errorf("Generation failed: %w", err)
		}

		if !outputToStdout {
			err := writeFile(cliOpts, bytes.NewReader(codeBuf.Bytes()), cliOpts.OutPath)
			if err != nil {
				return err
			}

			return nil
		}

		_, err = io.Copy(os.Stdout, codeBuf)
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}

		return nil
	}

	_, err = os.Stat(cliOpts.OutPath)
	if err != nil {
		if os.IsNotExist(err) {
			if !cliOpts.Yes {
				create := false
				prompt := &survey.Confirm{
					Message: fmt.Sprintf(`the directory "%v" doesn't exist, create it?`, cliOpts.OutPath),
				}
				err = survey.AskOne(prompt, &create)
				if err != nil {
					return err
				}
				if !create {
					return fmt.Errorf("aborted")
				}
			}
		} else {
			return fmt.Errorf("failed to stat target directory: %w", err)
		}
	}

	err = os.MkdirAll(cliOpts.OutPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if options.FilePattern == "" {
		options.FilePattern = "{{.Generator}}.gen.go"
	}

	separateTargets := len(hasTarget.FindStringIndex(options.FilePattern)) != 0
	if separateTargets && len(hasGenerator.FindStringIndex(options.FilePattern)) == 0 {
		return fmt.Errorf("generator must also be specified if target is specified in the file pattern")
	}

	for _, g := range generators {
		if separateTargets {
			for _, t := range options.Generators[g.Name()].Targets {
				fnBuf := &bytes.Buffer{}

				err = fileNameTemplate.Execute(fnBuf, &filenameValues{
					Generator: g.Name(),
					Target:    t,
				})
				if err != nil {
					return fmt.Errorf("invalid file pattern: %w", err)
				}

				fName := filepath.Join(cliOpts.OutPath, fnBuf.String())

				codeBuf := &bytes.Buffer{}

				err = generateUnit(
					ctx,
					options,
					spec,
					[]generator.Generator{g},
					map[string][]string{
						g.Name(): []string{t},
					},
					codeBuf,
				)
				if err != nil {
					return err
				}

				err := writeFile(cliOpts, bytes.NewReader(codeBuf.Bytes()), fName)
				if err != nil {
					return err
				}
			}

			continue
		}

		fnBuf := &bytes.Buffer{}

		err = fileNameTemplate.Execute(fnBuf, &filenameValues{
			Generator: g.Name(),
		})
		if err != nil {
			return fmt.Errorf("invalid file pattern: %w", err)
		}

		fName := filepath.Join(cliOpts.OutPath, fnBuf.String())

		codeBuf := &bytes.Buffer{}

		err = generateUnit(
			ctx,
			options,
			spec,
			[]generator.Generator{g},
			map[string][]string{
				g.Name(): options.Generators[g.Name()].Targets,
			},
			codeBuf,
		)
		if err != nil {
			return err
		}

		err := writeFile(cliOpts, bytes.NewReader(codeBuf.Bytes()), fName)
		if err != nil {
			return err
		}
	}

	return nil
}

// Essentially a single file
func generateUnit(
	ctx context.Context,
	options *config.ReposeOptions,
	spec *spec.Spec,
	generators []generator.Generator,
	targets map[string][]string,
	w io.Writer,
) error {
	codeBuf := &bytes.Buffer{}
	jenFile := jen.NewFile(options.PackageName)

	if options.Comments {
		if options.Timestamp {
			jenFile.HeaderComment(fmt.Sprintf("This code generated by Repose at %v.", time.Now().Format(time.RFC1123)))
		} else {
			jenFile.HeaderComment(fmt.Sprintf("This code generated by Repose."))
		}
	}

	for _, g := range generators {
		for _, t := range targets[g.Name()] {
			out, err := g.Generate(ctx, options.Generators[g.Name()].Options, spec, t)
			if err != nil {
				return fmt.Errorf("generator %v failed: %w", g.Name(), err)
			}

			switch c := out.(type) {
			case jen.Code:
				jenFile.Add(c)
			case []byte:
				codeBuf.Write(c)
				codeBuf.WriteString("\n")
			case string:
				codeBuf.WriteString(c + "\n")
			default:
				panic("generator gave wrong output: " + fmt.Sprint(c))
			}

			cli.Verbosef("Generating %v using %v.\n", t, g.Name())
		}
	}

	goCodeBuf := &bytes.Buffer{}

	for name, path := range ctx.Value(common.ContextState).(*common.State).PackageAliases() {
		jenFile.ImportAlias(path, name)
	}

	err := jenFile.Render(goCodeBuf)
	if err != nil {
		return fmt.Errorf("failed to render code: %w", err)
	}

	_, err = io.Copy(w, goCodeBuf)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	_, err = w.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	_, err = io.Copy(w, codeBuf)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}

func parseSpec(
	ctx context.Context,
	cliOpts *config.GenerateOptions,
	options *config.ReposeOptions,
	inPaths []string,
) (*spec.Spec, error) {
	if len(inPaths) == 0 {
		return nil, fmt.Errorf("no input specified")
	}

	inputFromStdin := len(inPaths) == 1 && inPaths[0] == "-"

	parsers, err := getParsers(options)
	if err != nil {
		return nil, err
	}

	if inputFromStdin {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from standard input %w", err)
		}

		errStrings := make([]string, 0, len(parsers))

		for _, p := range parsers {
			spec, err := p.Parse(ctx, options.Parsers[p.Name()], data)
			if err != nil {
				errStrings = append(errStrings, fmt.Sprintf("%v: %v", p.Name(), err.Error()))
				continue
			}

			cli.Successf("Specification was successfully parsed by the %v parser.\n", p.Name())

			return spec, nil
		}

		return nil, fmt.Errorf("no parsers could parse the data, parsers tried:\n%v", strings.Join(errStrings, "\n\n"))
	}

	filePaths := make([]string, 0)

	for _, inPath := range inPaths {
		err := filepath.Walk(inPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if path != inPath && !cliOpts.Recursive {
					return filepath.SkipDir
				}
				return nil
			}
			filePaths = append(filePaths, path)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to read files: %w", err)
		}
	}
	cli.Verbosef("Found %v files.\n", len(filePaths))

	errStrings := make([]string, 0, len(parsers))

	for _, p := range parsers {
		spec, err := p.ParseResources(ctx, options.Parsers[p.Name()], filePaths...)
		if err != nil {
			errStrings = append(errStrings, fmt.Sprintf("%v: %v", p.Name(), err.Error()))
			continue
		}

		cli.Successf("Specification was successfully parsed by the %v parser.\n", p.Name())
		return spec, nil
	}

	return nil, fmt.Errorf("no parsers could parse the input files, parsers tried:\n%v", strings.Join(errStrings, "\n\n"))
}

func normalizeNames(options *config.ReposeOptions) {
	for pName, pVal := range options.Parsers {
		normalizedName := strings.ToLower(strings.TrimSpace(pName))

		if normalizedName != pName {
			options.Parsers[normalizedName] = pVal
			delete(options.Parsers, pName)
		}
	}

	for gName, gVal := range options.Generators {
		normalizedName := strings.ToLower(strings.TrimSpace(gName))

		if normalizedName != gName {
			options.Generators[normalizedName] = gVal
			delete(options.Generators, gName)
		}
	}

	for _, transformer := range options.Transformers {
		transformer.Name = strings.ToLower(strings.TrimSpace(transformer.Name))
	}
}

func getParsers(options *config.ReposeOptions) ([]parser.Parser, error) {
	if len(options.Parsers) == 0 {
		return config.Parsers, nil
	}

	parsers := make([]parser.Parser, 0, len(options.Parsers))

	for pName := range options.Parsers {
		var found bool
		for _, parser := range config.Parsers {
			if parser.Name() == pName {
				parsers = append(parsers, parser)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(`parser with name "%v" not found`, pName)
		}
	}

	return parsers, nil
}

func getTransformers(options *config.ReposeOptions) ([]transformer.Transformer, error) {
	transformers := make([]transformer.Transformer, 0, len(options.Transformers))

	for _, tOption := range options.Transformers {
		var found bool
		for _, transformer := range config.Transformers {
			if transformer.Name() == tOption.Name {
				transformers = append(transformers, transformer)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(`transformer with name "%v" not found`, tOption.Name)
		}
	}

	return transformers, nil
}

func getGenerators(cliOpts *config.GenerateOptions, options *config.ReposeOptions) ([]generator.Generator, error) {
	generators := make([]generator.Generator, 0, len(options.Generators))

	if cliOpts.Targets != "" {

		cliGenerators := make(map[string][]string)

		for _, g := range strings.Split(cliOpts.Targets, ",") {
			gVals := strings.Split(g, ":")
			gName := gVals[0]
			if len(gVals) == 1 {
				return nil, fmt.Errorf("generator %v has no targets", gName)
			}

			if len(gVals) > 2 {
				return nil, fmt.Errorf("invalid format for generator %v", gName)
			}

			cliGenerators[gName] = strings.Split(gVals[1], "+")
		}

		for gName, gOpts := range options.Generators {
			if cliTargets, ok := cliGenerators[gName]; ok {
				gOpts.Targets = cliTargets
			} else {
				delete(options.Generators, gName)
			}
		}
	}

	for gName := range options.Generators {
		var found bool
		for _, generator := range config.Generators {
			if generator.Name() == gName {
				generators = append(generators, generator)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(`generator with name "%v" not found`, gName)
		}
	}

	return generators, nil
}

func isDir(p string) bool {
	if len(p) == 0 {
		return false
	}

	if p[len(p)-1] == '/' || p[len(p)-1] == '\\' {
		return true
	}

	fInfo, err := os.Stat(p)
	if err != nil {
		return false
	}

	return fInfo.IsDir()
}

func writeFile(cliOpts *config.GenerateOptions, codeBuf io.ReadSeeker, path string) error {
	if _, err := os.Stat(path); err == nil {
		if !cliOpts.Yes {
			cont := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf(`the file "%v" already exists, continue?`, path),
			}
			err = survey.AskOne(prompt, &cont)
			if err != nil {
				return err
			}
			if !cont {
				return fmt.Errorf("aborted")
			}
		}
	} else {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	existingFile := &bytes.Buffer{}
	_, err = io.Copy(existingFile, f)
	if err != nil {
		return fmt.Errorf("failed to read existing file %v: %w", f.Name(), err)
	}
	err = f.Close()
	if err != nil {
		return err
	}

	outBuf := &bytes.Buffer{}
	outOldBuf := &bytes.Buffer{}

	err = writeWithKeep(bytes.NewReader(existingFile.Bytes()), codeBuf, outBuf, outOldBuf)
	if err != nil {
		return err
	}

	newFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, outBuf)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	absName, err := filepath.Abs(path)
	if err != nil {
		absName = path
	}

	cli.Successf("%v written.\n", absName)

	if outOldBuf.Len() > 0 {
		postNum := 0
		for {
			bkFileNamePath := path + "old" + strconv.Itoa(postNum)
			_, err := os.Stat(bkFileNamePath)
			if err == nil {
				postNum++
				continue
			}

			newFile, err := os.Create(bkFileNamePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer newFile.Close()

			_, err = io.Copy(newFile, outOldBuf)
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}

			absName, err := filepath.Abs(bkFileNamePath)
			if err != nil {
				absName = path
			}

			cli.Successf("%v written.\n", absName)
		}
	}

	return nil
}

type keepBlock struct {
	tag  string
	code *bytes.Buffer
}

func writeWithKeep(existing, generated io.ReadSeeker, out, outOld io.Writer) error {
	exKeep, err := keepBlocks(existing)
	if err != nil {
		return err
	}

	genKeep, err := keepBlocks(generated)
	if err != nil {
		return err
	}
	_, err = generated.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	obsoleteKeep, err := diffKeepBlocks(genKeep, exKeep)
	if err != nil {
		return err
	}

	scn := bufio.NewScanner(generated)

	keepStart := regexp.MustCompile(`repose:keep\s([a-zA-Z0-9_]+)\s?\b`)

	keep := false
	for scn.Scan() {
		if !keep {
			foundTags := keepStart.FindStringSubmatch(scn.Text())

			if len(foundTags) != 0 {
				_, err = out.Write(genKeep[foundTags[1]].code.Bytes())
				if err != nil {
					return err
				}
				keep = true
				continue
			}

			_, err = out.Write(scn.Bytes())
			if err != nil {
				return err
			}
			_, err = out.Write([]byte("\n"))
			if err != nil {
				return err
			}
			continue
		}

		if bytes.Contains(scn.Bytes(), []byte("repose:endkeep")) {
			keep = false
			continue
		}
	}

	if len(obsoleteKeep) > 0 {
		for _, b := range obsoleteKeep {
			_, err = outOld.Write(b.code.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func diffKeepBlocks(targetKeep, oldKeep map[string]keepBlock) (map[string]keepBlock, error) {
	obsolete := make(map[string]keepBlock)

	newTags := make([]string, 0, len(targetKeep))

	for tag := range targetKeep {
		if oldBlock, ok := oldKeep[tag]; ok {
			targetKeep[tag] = oldBlock
			delete(oldKeep, tag)
			continue
		}

		newTags = append(newTags, tag)
	}

	for tag, block := range oldKeep {
		choiceIgnore := "ignore (default)"
		choiceBackup := "backup to a separate file"
		choiceNewTag := "choose new tag"

		keepChoices := []string{
			choiceIgnore,
		}

		if len(newTags) > 0 {
			keepChoices = append(keepChoices, choiceNewTag)
		}
		keepChoices = append(keepChoices, choiceBackup)

		cli.Warningf("Keep tag %v doesn't exist in the newly generated code.\n", tag)

		var choice string
		prompt := &survey.Select{
			Message: "What to do with the existing code?",
			Options: keepChoices,
		}
		err := survey.AskOne(prompt, &choice)
		if err != nil {
			return nil, err
		}

		switch choice {
		case choiceBackup:
			obsolete[tag] = block
		case choiceNewTag:
			var newTag string
			prompt := &survey.Select{
				Message: "Which new tag?",
				Options: newTags,
			}
			err = survey.AskOne(prompt, &newTag)
			if err != nil {
				return nil, err
			}

			for i := range newTags {
				if newTags[i] == newTag {
					newTags[i] = newTags[len(newTags)-1]
					newTags = newTags[:len(newTags)-1]
					break
				}
			}

			oldBytes := block.code.Bytes()

			newLineIdx := bytes.Index(oldBytes, []byte{'\n'})

			newNewFirstLineBytes := bytes.Replace(oldBytes[:newLineIdx], []byte(tag), []byte(newTag), 1)

			block.code.Reset()
			_, err = block.code.Write(append(newNewFirstLineBytes, oldBytes[newLineIdx:]...))
			if err != nil {
				return nil, err
			}

			targetKeep[newTag] = block
		}

	}

	return obsolete, nil
}

func keepBlocks(r io.Reader) (map[string]keepBlock, error) {
	keepStart := regexp.MustCompile(`repose:keep\s([a-zA-Z0-9_]+)\s?\b`)

	keep := make(map[string]keepBlock)

	scn := bufio.NewScanner(r)

	var currentKeep *keepBlock
	for scn.Scan() {
		foundTags := keepStart.FindStringSubmatch(scn.Text())
		if currentKeep == nil {
			if len(foundTags) == 0 {
				continue
			} else if len(foundTags) > 2 {
				return nil, fmt.Errorf("multiple keep tags on one line: \"%v\"", scn.Text())
			}

			currentKeep = &keepBlock{
				tag:  foundTags[1],
				code: new(bytes.Buffer),
			}

			currentKeep.code.Write(scn.Bytes())
			currentKeep.code.WriteByte('\n')
			continue
		}

		if len(foundTags) != 0 {
			return nil, fmt.Errorf("repose:keep %v has missing associated repose:endkeep", currentKeep.tag)
		}

		currentKeep.code.Write(scn.Bytes())
		currentKeep.code.WriteByte('\n')

		if bytes.Contains(scn.Bytes(), []byte("repose:endkeep")) {
			keep[currentKeep.tag] = *currentKeep
			currentKeep = nil
			continue
		}
	}

	if currentKeep != nil {
		return nil, fmt.Errorf("repose:keep %v has missing associated repose:endkeep", currentKeep.tag)
	}

	return keep, nil
}
