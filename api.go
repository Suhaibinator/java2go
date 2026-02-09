package java2go

import "github.com/NickyBoy89/java2go/transpiler"

// Run converts Java source inputs into Go output according to CLI-style flags.
func Run(args []string) error {
	return transpiler.Run(args)
}
