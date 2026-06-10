package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func convertJavaProjectDir(t *testing.T, root string) map[string]string {
	t.Helper()

	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() {
		symbol.GlobalScope = previousGlobal
	})

	files, err := parsing.ReadSourcesInDir(root)
	if err != nil {
		t.Fatalf("failed to collect java sources from %s: %v", root, err)
	}
	if len(files) == 0 {
		t.Fatalf("expected java sources under %s, found none", root)
	}

	for index := range files {
		if err := files[index].ParseAST(); err != nil {
			t.Fatalf("failed to parse AST for %s: %v", files[index].Name, err)
		}
	}

	for index := range files {
		if files[index].Ast.HasError() {
			t.Fatalf("parsed AST has errors for %s", files[index].Name)
		}
		symbols := files[index].ParseSymbols()
		symbol.AddSymbolsToPackage(symbols)
	}

	for _, file := range files {
		ResolveFile(file)
	}

	outputs := make(map[string]string, len(files))
	for _, file := range files {
		ctx := Ctx{
			currentFile:  file.Symbols,
			currentClass: file.Symbols.BaseClass,
		}
		parsed := ParseNode(file.Ast, file.Source, ctx).(ast.Node)

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, token.NewFileSet(), parsed); err != nil {
			t.Fatalf("failed to print generated Go for %s: %v", file.Name, err)
		}

		rel, err := filepath.Rel(root, file.Name)
		if err != nil {
			t.Fatalf("failed to create relative path for %s: %v", file.Name, err)
		}
		key := strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel)) + ".go"
		outputs[key] = buf.String()
	}

	return outputs
}

func TestFullProgram_MultiPackageConversion(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program")
	outputs := convertJavaProjectDir(t, root)

	requiredFiles := []string{
		"com/acme/common/Mapper.go",
		"com/acme/common/Mode.go",
		"com/acme/domain/ParseTask.go",
		"com/acme/app/Pipeline.go",
		"com/acme/app/MainApp.go",
	}
	for _, file := range requiredFiles {
		if _, exists := outputs[file]; !exists {
			t.Fatalf("expected generated output for %s; got keys: %v", file, outputs)
		}
	}

	mapperOut := normalizeSpaces(outputs["com/acme/common/Mapper.go"])
	if !strings.Contains(mapperOut, "type Mapper[T any, R any] interface") {
		t.Fatalf("expected generic mapper interface in output:\n%s", outputs["com/acme/common/Mapper.go"])
	}
	if !strings.Contains(mapperOut, "type MapperFuncAdapter[T any, R any] struct") {
		t.Fatalf("expected functional adapter for mapper interface:\n%s", outputs["com/acme/common/Mapper.go"])
	}

	loggerOut := normalizeSpaces(outputs["com/acme/common/Logger.go"])
	if !strings.Contains(loggerOut, "fmt.Println(msg)") {
		t.Fatalf("expected System.out.println lowering to fmt.Println in Logger:\n%s", outputs["com/acme/common/Logger.go"])
	}
	if !strings.Contains(loggerOut, "fmt \"fmt\"") {
		t.Fatalf("expected fmt import in Logger output:\n%s", outputs["com/acme/common/Logger.go"])
	}

	parseTaskOut := normalizeSpaces(outputs["com/acme/domain/ParseTask.go"])
	if !strings.Contains(parseTaskOut, "common.NewMapperFuncAdapter[string, string]") {
		t.Fatalf("expected package-qualified lambda wrapper call in ParseTask:\n%s", outputs["com/acme/domain/ParseTask.go"])
	}
	if !strings.Contains(parseTaskOut, "any(normalized).(string)") {
		t.Fatalf("expected instanceof conversion in ParseTask:\n%s", outputs["com/acme/domain/ParseTask.go"])
	}
	if !strings.Contains(parseTaskOut, "stdjava.StringLength(normalized)") {
		t.Fatalf("expected String.length() lowering in ParseTask:\n%s", outputs["com/acme/domain/ParseTask.go"])
	}
	if !strings.Contains(parseTaskOut, "common \"com/acme/common\"") {
		t.Fatalf("expected generated import for referenced common package in ParseTask:\n%s", outputs["com/acme/domain/ParseTask.go"])
	}

	pipelineOut := normalizeSpaces(outputs["com/acme/app/Pipeline.go"])
	if !strings.Contains(pipelineOut, "func Execute(task domain.TaskI, mapper common.Mapper[string, string]) int32") {
		t.Fatalf("expected abstract-class interface parameter in Pipeline:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "any(task).(*domain.ParseTask)") {
		t.Fatalf("expected package-qualified class instanceof conversion in Pipeline:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "stdjava.StringLength(out)") {
		t.Fatalf("expected String.length() lowering in Pipeline:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "common \"com/acme/common\"") || !strings.Contains(pipelineOut, "domain \"com/acme/domain\"") {
		t.Fatalf("expected generated imports for referenced cross-package types in Pipeline:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "recover(") {
		t.Fatalf("expected try/catch lowering in Pipeline guardedValue:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "50") || !strings.Contains(pipelineOut, "3") {
		t.Fatalf("expected catch/finally constants in Pipeline guardedValue:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "func GuardedFinallyOverride() int32") {
		t.Fatalf("expected guardedFinallyOverride method to be converted:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "func GuardedCatchFinallyOverride() int32") {
		t.Fatalf("expected guardedCatchFinallyOverride method to be converted:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "func GuardedFinallyPanicOverride() int32") {
		t.Fatalf("expected guardedFinallyPanicOverride method to be converted:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "func GuardedResourceOrder() string") {
		t.Fatalf("expected guardedResourceOrder method to be converted:\n%s", outputs["com/acme/app/Pipeline.go"])
	}
	if !strings.Contains(pipelineOut, "defer func()") || !strings.Contains(pipelineOut, ".Close()") {
		t.Fatalf("expected try-with-resources close lowering using defer and Close calls:\n%s", outputs["com/acme/app/Pipeline.go"])
	}

	mainOut := normalizeSpaces(outputs["com/acme/app/MainApp.go"])
	if !strings.Contains(mainOut, "domain.NewParseTask(\"alpha\")") {
		t.Fatalf("expected package-qualified constructor call in MainApp:\n%s", outputs["com/acme/app/MainApp.go"])
	}
	if !strings.Contains(mainOut, "common.ModeValueOf(\"FAST\")") {
		t.Fatalf("expected enum valueOf helper usage in MainApp:\n%s", outputs["com/acme/app/MainApp.go"])
	}
	if !strings.Contains(mainOut, "common.ModeValues()") {
		t.Fatalf("expected enum values helper usage in MainApp:\n%s", outputs["com/acme/app/MainApp.go"])
	}
	if !strings.Contains(mainOut, "common \"com/acme/common\"") || !strings.Contains(mainOut, "domain \"com/acme/domain\"") {
		t.Fatalf("expected generated imports for referenced packages in MainApp:\n%s", outputs["com/acme/app/MainApp.go"])
	}
	if !strings.Contains(mainOut, "Pipeline.Execute") && !strings.Contains(mainOut, "Execute(") {
		t.Fatalf("expected pipeline execution call in MainApp:\n%s", outputs["com/acme/app/MainApp.go"])
	}
}
