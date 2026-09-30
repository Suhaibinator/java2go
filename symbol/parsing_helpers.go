package symbol

import (
	"bytes"
	"go/printer"
	"go/token"
	"unicode"
	"unicode/utf8"
)

// Uppercase uppercases the first character of the given string
func Uppercase(name string) string {
	if name == "" {
		return name
	}
	first, width := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(first)) + name[width:]
}

// Lowercase lowercases the first character of the given string
func Lowercase(name string) string {
	if name == "" {
		return name
	}
	first, width := utf8.DecodeRuneInString(name)
	return string(unicode.ToLower(first)) + name[width:]
}

// HandleExportStatus is a convenience method for renaming methods that may be
// either public or private, and need to be renamed
func HandleExportStatus(exported bool, name string) string {
	if dollarName := dollarIdentifierVisibility(exported, name); dollarName != "" {
		return dollarName
	}
	if exported {
		return Uppercase(name)
	}
	return Lowercase(name)
}

func classIdentifier(exported bool, name string) string {
	name = HandleExportStatus(exported, name)
	// Visibility conversion can itself create a Go keyword, for example a
	// package-private Java class named If. Escape its generated declaration.
	if token.Lookup(name).IsKeyword() {
		name += "_"
	}
	return name
}

// nodeToStr converts any AST node to its string representation
func nodeToStr(node any) string {
	var s bytes.Buffer
	err := printer.Fprint(&s, token.NewFileSet(), node)
	if err != nil {
		panic(err)
	}
	return s.String()
}

func NodeToStr(node any) string {
	return nodeToStr(node)
}
