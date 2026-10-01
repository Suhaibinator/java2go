package parsing

import (
	"context"
	"fmt"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

type SourceFile struct {
	Name   string
	Source []byte
	// UnicodeSource maps parser byte offsets back to the unchanged input when
	// repeated-u Unicode escape spellings required grammar canonicalization.
	UnicodeSource *UnicodeSourceMap
	Ast           *sitter.Node
	Symbols       *symbol.FileScope
}

func (file SourceFile) String() string {
	return fmt.Sprintf("SourceFile { Name: %s, Ast: %v, Symbols: %v }", file.Name, file.Ast, file.Symbols)
}

func (file *SourceFile) ParseAST() error {
	file.canonicalizeUnicodeEscapes()
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, file.Source)
	if err != nil {
		return err
	}

	if parserSource, repairs := parenthesizedAssignmentParserSource(tree.RootNode(), file.Source); len(repairs) != 0 {
		repaired, repairError := parser.ParseCtx(context.Background(), nil, parserSource)
		if repairError != nil {
			return repairError
		}
		if parenthesizedAssignmentsRecovered(repaired.RootNode(), file.Source, repairs) {
			tree = repaired
		}
	}
	file.Ast = tree.RootNode()
	return nil
}

func (file *SourceFile) ParseSymbols() *symbol.FileScope {
	symbols := symbol.ParseSymbols(file.Ast, file.Source)
	if file.UnicodeSource != nil {
		symbols.OriginalSource = file.UnicodeSource.Original
		symbols.SourceOffsets = append([]uint32(nil), file.UnicodeSource.offsets...)
	}
	file.Symbols = symbols
	return symbols
}
