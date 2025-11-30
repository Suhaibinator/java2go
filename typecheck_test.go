package main

import (
	"context"
	"os"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func loadFile(fileName string) ([]byte, *sitter.Tree) {
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	source, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		panic(err)
	}
	return source, tree
}

// TODO: TypeInformation and ExtractTypeInformation need to be implemented
// These tests are skipped until the type checking functionality is added

func TestSimpleDeclaration(t *testing.T) {
	t.Skip("TypeInformation and ExtractTypeInformation not yet implemented")
}

func TestMethodDeclaration(t *testing.T) {
	t.Skip("TypeInformation and ExtractTypeInformation not yet implemented")
}
