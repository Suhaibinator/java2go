package java2go

import "github.com/NickyBoy89/java2go/transpiler"

// Diagnostic describes a single unsupported Java construct encountered during
// conversion. See transpiler.Diagnostic for details.
type Diagnostic = transpiler.Diagnostic

// Run converts Java source inputs into Go output according to CLI-style flags.
//
// By default, unsupported constructs are converted into `// UNSUPPORTED:` stubs
// and recorded as diagnostics instead of aborting the conversion. Pass the
// "-strict" flag in args to restore fail-fast behavior, where the first
// unsupported construct returns an error.
func Run(args []string) error {
	return transpiler.Run(args)
}

// Diagnostics returns the unsupported-construct diagnostics gathered by the most
// recent Run. Each call to Run resets the collected diagnostics.
func Diagnostics() []Diagnostic {
	return transpiler.Diagnostics()
}
