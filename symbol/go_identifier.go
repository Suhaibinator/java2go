package symbol

import (
	"encoding/hex"
	"strings"
	"unicode"
	"unicode/utf8"
)

const goIdentifierEscape = "Java2goIdentifier_"

// GoIdentifier maps a completed generated name into Go's identifier alphabet.
// Java lookup and nominal metadata continue to use the original Java spelling.
// The escape namespace is escaped too: a legal Java name spelled like the
// encoding of a dollar-bearing name must never capture that declaration.
// This is a one-time emission boundary, not an idempotent symbol rename.
func GoIdentifier(name string) string {
	if !strings.Contains(name, "$") && !strings.Contains(name, goIdentifierEscape) && !strings.Contains(name, Lowercase(goIdentifierEscape)) {
		return name
	}
	prefix := Lowercase(goIdentifierEscape)
	first, _ := utf8.DecodeRuneInString(name)
	if unicode.IsUpper(first) {
		prefix = goIdentifierEscape
	}
	return prefix + hex.EncodeToString([]byte(name))
}

// Keep the complete source spelling of every dollar-bearing declaration before
// visibility conversion. Java small$ and Small$ are distinct, even when both
// declarations have the same visibility. The slash is impossible in a Java
// identifier, so these internal spellings cannot collide with source names.
func dollarIdentifierVisibility(exported bool, name string) string {
	if !strings.Contains(name, "$") {
		return ""
	}
	if exported {
		return ExportedGoBinding(name)
	}
	return "java2go$Private/" + name
}

// ExportedGoBinding preserves already exported spellings. For implicit-public
// Java declarations such as enum constants, capitalizing the source name would
// merge distinct names (small and Small). The internal marker preserves their
// spelling and their export status until the one-time emission boundary.
func ExportedGoBinding(name string) string {
	first, _ := utf8.DecodeRuneInString(name)
	if !strings.Contains(name, "$") && unicode.IsUpper(first) {
		return name
	}
	return "Java2go$Public/" + name
}
