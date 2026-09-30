package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
)

// Non-generic abstract references use companion interfaces. Generic abstract
// references retain a physical *Base[T] view, whose dispatch field points at the
// most-derived receiver; invoking the base's method directly would hit its stub.
func abstractClassUsesInterfaceView(scope *symbol.ClassScope) bool {
	return scope != nil && scope.IsAbstract && len(scope.TypeParameters) == 0
}
