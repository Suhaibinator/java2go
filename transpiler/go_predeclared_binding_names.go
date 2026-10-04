package transpiler

import "go/types"

// goPredeclaredBindingNames returns the Go universe bindings which generated
// package declarations must not capture. It contains no Java source-name policy.
func goPredeclaredBindingNames() []string {
	return types.Universe.Names()
}
