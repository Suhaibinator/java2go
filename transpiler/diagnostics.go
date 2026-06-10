package transpiler

import (
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

// Diagnostic records a single unsupported construct encountered while converting
// a file. Instead of aborting the whole conversion, the transpiler emits a
// diagnostic and falls back to an `// UNSUPPORTED:` stub so the rest of the file
// can still be converted.
type Diagnostic struct {
	// Kind describes the conversion phase that failed, e.g. "expression",
	// "statement", "declaration", or "node".
	Kind string
	// NodeType is the tree-sitter node type that could not be converted.
	NodeType string
	// Message is a human-readable description of the problem.
	Message string
	// Line is the 1-based source line where the construct begins, or 0 if unknown.
	Line uint32
}

func (d Diagnostic) String() string {
	if d.Line > 0 {
		return fmt.Sprintf("line %d: unsupported %s %q: %s", d.Line, d.Kind, d.NodeType, d.Message)
	}
	return fmt.Sprintf("unsupported %s %q: %s", d.Kind, d.NodeType, d.Message)
}

// diagnostics collects unsupported-construct reports for the current conversion.
// Conversion of a single file is sequential, but parsing happens in parallel, so
// the collector is guarded by a mutex to stay safe under concurrent use.
var diagnostics struct {
	mu     sync.Mutex
	items  []Diagnostic
	strict bool
}

// strictModeError is returned (via panic/recover at the top of conversion) when a
// diagnostic is reported while strict mode is enabled, restoring the original
// fail-fast behavior.
type strictModeError struct {
	diagnostic Diagnostic
}

func (e strictModeError) Error() string {
	return e.diagnostic.String()
}

// setStrictMode toggles fail-fast behavior. When enabled, the first unsupported
// construct aborts conversion instead of emitting a stub.
func setStrictMode(strict bool) {
	diagnostics.mu.Lock()
	defer diagnostics.mu.Unlock()
	diagnostics.strict = strict
}

// resetDiagnostics clears any previously collected diagnostics. It is called at
// the start of each conversion run.
func resetDiagnostics() {
	diagnostics.mu.Lock()
	defer diagnostics.mu.Unlock()
	diagnostics.items = nil
}

// Diagnostics returns the unsupported-construct diagnostics gathered by the most
// recent conversion run. It is part of the library API so callers can inspect
// what could not be converted without parsing log output.
func Diagnostics() []Diagnostic {
	return collectedDiagnostics()
}

// collectedDiagnostics returns a copy of the diagnostics gathered so far.
func collectedDiagnostics() []Diagnostic {
	diagnostics.mu.Lock()
	defer diagnostics.mu.Unlock()
	out := make([]Diagnostic, len(diagnostics.items))
	copy(out, diagnostics.items)
	return out
}

// reportUnsupported records a diagnostic for an unsupported construct. In strict
// mode it panics with a strictModeError, which the top-level conversion recovers
// into a returned error.
func reportUnsupported(kind string, node *sitter.Node, source []byte, ctx Ctx) Diagnostic {
	diag := Diagnostic{
		Kind: kind,
	}
	if node != nil {
		diag.NodeType = node.Type()
		diag.Line = node.StartPoint().Row + 1
		diag.Message = nodeSnippet(node, source)
	}

	diagnostics.mu.Lock()
	strict := diagnostics.strict
	diagnostics.items = append(diagnostics.items, diag)
	diagnostics.mu.Unlock()

	if strict {
		panic(strictModeError{diagnostic: diag})
	}

	return diag
}

// nodeSnippet returns a short, single-line description of a node's source text,
// suitable for inclusion in a diagnostic message or comment stub.
func nodeSnippet(node *sitter.Node, source []byte) string {
	if node == nil || source == nil {
		return ""
	}
	content := node.Content(source)
	// Collapse to a single line so it can live inside a `//` comment.
	const maxLen = 80
	trimmed := make([]rune, 0, len(content))
	for _, r := range content {
		switch r {
		case '\n', '\r', '\t':
			r = ' '
		}
		trimmed = append(trimmed, r)
	}
	// Truncate on the rune slice, not the string, so a multi-byte rune is never
	// split into invalid UTF-8.
	if len(trimmed) > maxLen {
		return string(trimmed[:maxLen]) + "..."
	}
	return string(trimmed)
}

// unsupportedComment builds the text of an `// UNSUPPORTED:` comment describing a
// diagnostic.
func unsupportedComment(diag Diagnostic) string {
	if diag.Message != "" {
		return fmt.Sprintf("// UNSUPPORTED: %s %q: %s", diag.Kind, diag.NodeType, diag.Message)
	}
	return fmt.Sprintf("// UNSUPPORTED: %s %q", diag.Kind, diag.NodeType)
}
