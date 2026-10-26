package transpiler

// A single fixed primitive declaration determines the conversion boundary of
// a static method reference. SAM parameters may be narrower or boxed. Other
// signature families keep their existing overload and generic resolution.
func staticPrimitiveReferenceParameters(class, method string, arity int) ([]string, bool) {
	definitions := staticIntrinsicImportSignatures[intrinsicKey{class, method}]
	if len(definitions) != 1 || definitions[0] == nil {
		return nil, false
	}
	definition := definitions[0]
	if !definition.IsStatic || definition.IsVariadic || len(definition.TypeParameters) != 0 || len(definition.Parameters) != arity {
		return nil, false
	}
	parameters := make([]string, arity)
	for index, parameter := range definition.Parameters {
		if parameter == nil || executionParameterIsVariadic(definition, index) || parameter.OriginalType == "void" {
			return nil, false
		}
		if _, primitive := javaPrimitiveType(parameter.OriginalType); !primitive {
			return nil, false
		}
		parameters[index] = parameter.OriginalType
	}
	return parameters, true
}
