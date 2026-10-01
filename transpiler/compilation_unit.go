package transpiler

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/parsing"
)

// compilationUnitInfo distinguishes executable type declarations from package
// metadata by syntax. Source filenames are not evidence of either kind.
type compilationUnitInfo struct {
	packageMetadata bool
	packageName     string
}

func classifyCompilationUnit(file parsing.SourceFile) (compilationUnitInfo, error) {
	if file.Ast == nil {
		return compilationUnitInfo{}, fmt.Errorf("missing Java AST in %s", file.Name)
	}
	if file.Ast.HasError() {
		return compilationUnitInfo{}, fmt.Errorf("java parse error in %s", file.Name)
	}
	var unit compilationUnitInfo
	types, imports := 0, 0
	for index := 0; index < int(file.Ast.NamedChildCount()); index++ {
		node := file.Ast.NamedChild(index)
		switch node.Type() {
		case "line_comment", "block_comment":
		case "package_declaration":
			// Package annotations precede the qualified name among named children.
			// Selecting the name by its syntax keeps annotations out of package identity.
			for child := 0; child < int(node.NamedChildCount()); child++ {
				name := node.NamedChild(child)
				if name.Type() == "identifier" || name.Type() == "scoped_identifier" {
					unit.packageName = name.Content(file.Source)
				}
			}
		case "import_declaration":
			imports++
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			types++
		default:
			return compilationUnitInfo{}, fmt.Errorf("unsupported compilation unit %q in %s at line %d", node.Type(), file.Name, node.StartPoint().Row+1)
		}
	}
	if types > 0 {
		return unit, nil
	}
	if unit.packageName != "" {
		unit.packageMetadata = true
		return unit, nil
	}
	if imports > 0 {
		return compilationUnitInfo{}, fmt.Errorf("no type or package declaration in Java compilation unit %s", file.Name)
	}
	return compilationUnitInfo{}, fmt.Errorf("empty Java compilation unit in %s", file.Name)
}

func packageMetadataGoFile(file parsing.SourceFile, unit compilationUnitInfo, ctx Ctx) *ast.File {
	name := unit.packageName
	if last := strings.LastIndexByte(name, '.'); last >= 0 {
		name = name[last+1:]
	}
	// Retain each metadata unit as a generated package-only source, including its
	// package annotations and documentation. They do not declare executable types.
	// This is source retention, not an implementation of Java package reflection.
	source := file.Source
	if file.UnicodeSource != nil {
		source = file.UnicodeSource.Original
	}
	comments := []*ast.Comment{{Text: "// Java package metadata (source declarations retained)"}}
	for _, line := range strings.Split(string(source), "\n") {
		comments = append(comments, &ast.Comment{Text: "// " + strings.TrimSuffix(line, "\r")})
	}
	program := &ast.File{Name: ast.NewIdent(name), Doc: &ast.CommentGroup{List: comments}}
	lowerGeneratedGoIdentifiers(program, ctx)
	return program
}
