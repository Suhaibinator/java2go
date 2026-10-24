package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Java collection interfaces can retain a concrete source/anonymous Go pointer
// at an expression. Different physical pointers (or class-subobject aliases)
// must still use the Java allocation's identity. This proof is declaration-based
// and intentionally does not extend admission to arrays, Map, or Go method shape.
func abstractCollectionReferenceComparison(javaType string, ctx Ctx) bool {
	component, rank := javaArrayTypeParts(javaType)
	if rank != 0 {
		return false
	}
	if _, primitive := javaPrimitiveType(component); primitive {
		return false
	}
	switch canonicalAbstractCollectionOwner(component, ctx) {
	case "java.util.Collection", "java.util.Set", "java.util.AbstractCollection", "java.util.AbstractSet":
		return true
	}
	base, _ := parseJavaTypeString(component)
	if _, binder := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); binder {
		return false
	}
	return sourceHasCollectionContract(resolveClassScopeByQualifiedName(ctx, base), ctx, map[*symbol.ClassScope]bool{})
}
