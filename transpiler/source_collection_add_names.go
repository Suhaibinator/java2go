package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// An ordinary Java CollectionAdd member would otherwise acquire the certified
// CollectionAddJava2goExecution selector. Reserve the entire execution prefix
// consistently while preserving the member's normal Java selector and calls.
func sourceCollectionAddImplementationName(method *symbol.Definition, owner *symbol.ClassScope) string {
	if method == nil || !strings.HasPrefix(method.Name, "CollectionAdd") {
		return ""
	}
	if strings.HasPrefix(method.Name+executionMethodSuffix, "CollectionAddJava2goExecution") {
		return collisionSafeExecutionIdentifier(method.Name+"Java2goSourceBodyExecution", owner)
	}
	return ""
}
