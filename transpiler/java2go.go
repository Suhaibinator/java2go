package transpiler

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

// Stores a global list of Java annotations to exclude from the generated code
var excludedAnnotations = make(map[string]bool)

func Run(args []string) error {
	return run(args, os.Stdout)
}

func run(args []string, stdout io.Writer) error {
	var writeFiles bool
	var dryRun bool
	var displayAST bool
	var symbolAware bool
	var parseFilesSynchronously bool
	var initGoMod bool
	var outputDirectory string
	var modulePath string
	var ignoredAnnotations string
	var inputRoots []string

	flagSet := flag.NewFlagSet("java2go", flag.ContinueOnError)
	flagSet.BoolVar(&writeFiles, "w", false, "Whether to write the files to disk instead of stdout")
	flagSet.BoolVar(&dryRun, "q", false, "Don't write to stdout on successful parse")
	flagSet.BoolVar(&displayAST, "ast", false, "Print out go's pretty-printed ast, instead of source code")
	flagSet.BoolVar(&parseFilesSynchronously, "sync", false, "Parse the files one by one, instead of in parallel")
	flagSet.BoolVar(&initGoMod, "init-go-mod", false, "Create a go.mod file in the output directory when writing files")
	flagSet.BoolVar(&symbolAware, "symbols", true, `Whether the program is aware of the symbols of the parsed code
Results in better code generation, but can be disabled for a more direct translation
or to fix crashes with the symbol handling`,
	)
	flagSet.StringVar(&outputDirectory, "output", ".", "Specify a directory for the generated files")
	flagSet.StringVar(&modulePath, "module", "generated", "Module path to use when creating go.mod")
	flagSet.StringVar(&ignoredAnnotations, "exclude-annotations", "", "A comma-separated list of annotations to exclude from the final code generation")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	excludedAnnotations = make(map[string]bool)
	for _, annotation := range strings.Split(ignoredAnnotations, ",") {
		annotation = strings.TrimSpace(annotation)
		if annotation == "" {
			continue
		}
		excludedAnnotations[annotation] = true
	}

	// All the files to parse
	var files []parsing.SourceFile

	log.Info("Collecting files...")

	// Collect all the files and read them into memory
	for _, dirName := range flagSet.Args() {
		inputRoots = append(inputRoots, inputRootForArg(dirName))
		sources, err := parsing.ReadSourcesInDir(dirName)
		if err != nil {
			return fmt.Errorf("error reading directory %s: %w", dirName, err)
		}
		files = append(files, sources...)
	}

	if len(files) == 0 {
		log.Warn("No files specified to convert")
	}

	// Parse the ASTs of all the files

	log.Info("Parsing ASTs...")

	var wg sync.WaitGroup
	wg.Add(len(files))
	parseErrors := make(chan error, len(files))

	for index := range files {
		parseFunc := func(ind int) {
			if err := files[ind].ParseAST(); err != nil {
				parseErrors <- fmt.Errorf("error parsing AST for %s: %w", files[ind].Name, err)
			}
			wg.Done()
		}

		if parseFilesSynchronously {
			parseFunc(index)
		} else {
			go parseFunc(index)
		}
	}

	// We might still have some parsing jobs, so wait on them
	wg.Wait()
	close(parseErrors)
	if firstErr := <-parseErrors; firstErr != nil {
		return firstErr
	}

	for _, file := range files {
		if file.Ast == nil {
			return errors.New("not all files have ASTs")
		}
	}

	if writeFiles && initGoMod {
		if err := ensureGoModFile(outputDirectory, modulePath); err != nil {
			return err
		}
	}

	// Generate the symbol tables for the files
	if symbolAware {
		log.Info("Generating symbol tables...")

		for index, file := range files {
			if file.Ast.HasError() {
				log.WithFields(log.Fields{
					"fileName": file.Name,
				}).Warn("AST parse error in file, skipping file")
				continue
			}

			symbols := files[index].ParseSymbols()
			// Add the symbols to the global symbol table
			symbol.AddSymbolsToPackage(symbols)
		}

		// Go back through the symbol tables and fill in anything that could not be resolved

		log.Info("Resolving symbols...")

		for _, file := range files {
			if !file.Ast.HasError() {
				ResolveFile(file)
			}
		}
	}

	// Transpile the files

	log.Info("Converting files...")

	for _, file := range files {
		if dryRun {
			log.Infof("Not converting file \"%s\"", file.Name)
			continue
		}

		log.Infof("Converting file \"%s\"", file.Name)

		// Write to stdout by default
		output := stdout
		if writeFiles {
			// Write to a `.go` file in the same directory
			outputFile := filepath.Join(outputDirectory, outputRelativePath(file, inputRoots, modulePath))

			err := os.MkdirAll(filepath.Dir(outputFile), 0755)
			if err != nil {
				return fmt.Errorf("error creating output directory for %s: %w", outputFile, err)
			}

			// Write the output to a file
			output, err = os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("error creating output file %s: %w", outputFile, err)
			}
		}

		// The converted AST, in Go's AST representation
		var initialContext Ctx
		if symbolAware {
			initialContext.currentFile = file.Symbols
			initialContext.currentClass = file.Symbols.BaseClass
		}

		parsed := ParseNode(file.Ast, file.Source, initialContext).(ast.Node)

		// Print the generated AST
		if displayAST {
			if err := ast.Print(token.NewFileSet(), parsed); err != nil {
				return fmt.Errorf("error printing generated AST for %s: %w", file.Name, err)
			}
		}

		// Output the parsed AST, into the source specified earlier
		if err := printer.Fprint(output, token.NewFileSet(), parsed); err != nil {
			return fmt.Errorf("error printing generated code for %s: %w", file.Name, err)
		}

		if writeFiles {
			if err := output.(*os.File).Close(); err != nil {
				return fmt.Errorf("error closing output file for %s: %w", file.Name, err)
			}
		}
	}

	return nil
}

func inputRootForArg(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return absPath
	}
	if info.IsDir() {
		return absPath
	}
	return filepath.Dir(absPath)
}

func modulePathToJavaPackage(modulePath string) string {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return ""
	}
	return strings.ReplaceAll(modulePath, "/", ".")
}

func packageRelativeToModule(javaPackage string, modulePath string) (string, bool) {
	javaPackage = strings.TrimSpace(javaPackage)
	modulePackage := modulePathToJavaPackage(modulePath)
	if javaPackage == "" || modulePackage == "" {
		return "", false
	}
	if javaPackage == modulePackage {
		return "", true
	}
	prefix := modulePackage + "."
	if strings.HasPrefix(javaPackage, prefix) {
		return strings.TrimPrefix(javaPackage, prefix), true
	}
	return "", false
}

func bestInputRootForFile(fileName string, inputRoots []string) string {
	absFile, err := filepath.Abs(fileName)
	if err != nil {
		return ""
	}

	best := ""
	for _, root := range inputRoots {
		root = filepath.Clean(root)
		if absFile == root || strings.HasPrefix(absFile, root+string(filepath.Separator)) {
			if len(root) > len(best) {
				best = root
			}
		}
	}
	return best
}

func outputRelativePath(file parsing.SourceFile, inputRoots []string, modulePath string) string {
	baseFile := strings.TrimSuffix(filepath.Base(file.Name), filepath.Ext(file.Name)) + ".go"

	if file.Symbols != nil {
		if packageSuffix, ok := packageRelativeToModule(file.Symbols.Package, modulePath); ok {
			if packageSuffix == "" {
				return baseFile
			}
			return filepath.Join(filepath.FromSlash(strings.ReplaceAll(packageSuffix, ".", "/")), baseFile)
		}
	}

	if root := bestInputRootForFile(file.Name, inputRoots); root != "" {
		absFile, err := filepath.Abs(file.Name)
		if err == nil {
			if rel, err := filepath.Rel(root, absFile); err == nil {
				return strings.TrimSuffix(rel, filepath.Ext(rel)) + ".go"
			}
		}
	}

	fallback := strings.TrimSuffix(filepath.Clean(file.Name), filepath.Ext(file.Name)) + ".go"
	// Avoid absolute paths under output directory in fallback.
	fallback = strings.TrimPrefix(fallback, filepath.VolumeName(fallback))
	fallback = strings.TrimPrefix(fallback, string(filepath.Separator))
	return fallback
}

func ensureGoModFile(outputDirectory string, modulePath string) error {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return errors.New("module path cannot be empty when creating go.mod")
	}

	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return fmt.Errorf("error creating output directory %s: %w", outputDirectory, err)
	}

	goModPath := filepath.Join(outputDirectory, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error checking go.mod at %s: %w", goModPath, err)
	}

	content := fmt.Sprintf("module %s\n\ngo 1.25.0\n", modulePath)
	if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing go.mod at %s: %w", goModPath, err)
	}

	return nil
}
