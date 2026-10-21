package transpiler

// The selected intrinsic signature owns primitive components independently of
// the caller's type namespace. Reference spellings retain their existing path.
func recordIntrinsicPrimitiveResultOrigin(outputs []*inferredJavaTypeOrigin, result string) {
	base, rank := javaArrayTypeParts(result)
	if _, primitive := javaPrimitiveType(base); !primitive {
		return
	}
	setInferredJavaTypeOrigin(outputs, inferredJavaTypeOrigin{javaType: result, arrayRank: rank})
}

// Both names have already been certified as canonical declaration origins.
// Consult only the existing Throwable graph, never caller spelling lookup.
func canonicalThrowableOriginAssignable(actual, formal inferredJavaTypeOrigin) bool {
	scalar := func(value inferredJavaTypeOrigin) bool {
		_, writtenRank := javaArrayTypeParts(value.javaType)
		return !value.unresolved && value.externalName != "" && value.nominalScope == nil && value.parameter == nil && value.arrayRank == 0 && writtenRank == 0 && !value.wildcard && len(value.arguments) == 0
	}
	if !scalar(actual) || !scalar(formal) {
		return false
	}
	// Require both endpoints to belong to the modeled canonical graph.
	if _, known := builtinThrowableReferenceParent(formal.externalName); !known {
		return false
	}
	seen := map[string]bool{}
	for owner := actual.externalName; owner != "" && !seen[owner]; {
		seen[owner] = true
		parent, known := builtinThrowableReferenceParent(owner)
		if !known {
			return false
		}
		if parent == formal.externalName {
			return true
		}
		owner = parent
	}
	return false
}
