package transpiler

import (
	"go/ast"
	"strings"
)

// Annotation source is documentation, never Go syntax. Give each physical line
// its own line comment, including text blocks and embedded Java comments. A
// space before continuation text prevents data such as go:linkname or line
// directives from acquiring compiler meaning. Normalize Java's three line
// terminators without changing the annotation's remaining contents.
func javaAnnotationComments(source string) []*ast.Comment {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	lines := strings.Split(source, "\n")
	comments := make([]*ast.Comment, len(lines))
	for index, line := range lines {
		prefix := "// "
		if index == 0 {
			// The first line begins with @, preserving existing single-line output.
			prefix = "//"
		}
		comments[index] = &ast.Comment{Text: prefix + line}
	}
	return comments
}
