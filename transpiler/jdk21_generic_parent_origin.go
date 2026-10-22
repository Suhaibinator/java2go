package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Parent recipes refer to the parameters of their actual native declaration.
// Tokens are private metadata identities; invocation binders arrive only as
// unchanged supplied origins during substitution.
type jdk21GenericParentRecord struct {
	certificate string
	parameters  []*symbol.TypeParamDeclaration
	parents     []inferredJavaTypeOrigin
}

func projectJDK21GenericParentOrigin(actual inferredJavaTypeOrigin, requestedOwner string) (inferredJavaTypeOrigin, bool) {
	return projectJDK21GenericParentWithRecords(actual, requestedOwner, jdk21GenericParentRecords, jdk21GenericParentCertificates)
}

func projectJDK21GenericParentWithRecords(actual inferredJavaTypeOrigin, requestedOwner string, records map[string]jdk21GenericParentRecord, certificates map[string]string) (inferredJavaTypeOrigin, bool) {
	if actual.nominalScope != nil || actual.parameter != nil || actual.externalName == "" || actual.arrayRank != 0 || actual.wildcard || !completeJDK21GenericOrigin(actual) {
		return inferredJavaTypeOrigin{}, false
	}
	// Nongeneric terminal records close side edges; they do not admit new targets.
	target, present := records[requestedOwner]
	if !present || len(target.parameters) == 0 {
		return inferredJavaTypeOrigin{}, false
	}
	var result inferredJavaTypeOrigin
	found := false
	var visit func(inferredJavaTypeOrigin, map[string]bool) bool
	visit = func(view inferredJavaTypeOrigin, path map[string]bool) bool {
		record, known := records[view.externalName]
		if !known || record.certificate == "" || record.certificate != certificates[view.externalName] || path[view.externalName] || view.arrayRank != 0 || view.wildcard || view.nominalScope != nil || view.parameter != nil {
			return false
		}
		owned := map[*symbol.TypeParamDeclaration]bool{}
		for _, parameter := range record.parameters {
			if parameter == nil || owned[parameter] {
				return false
			}
			owned[parameter] = true
		}
		raw := len(record.parameters) > 0 && len(view.arguments) == 0
		if !raw && len(view.arguments) != len(record.parameters) {
			return false
		}
		if len(record.parameters) == 0 && len(view.arguments) != 0 {
			return false
		}
		if view.externalName == requestedOwner {
			if found && !sameJavaTypeOrigin(result, view) {
				return false
			}
			result, found = view, true
		}
		next := make(map[string]bool, len(path)+1)
		for owner, present := range path {
			next[owner] = present
		}
		next[view.externalName] = true
		bindings := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
		if !raw {
			for index, parameter := range record.parameters {
				bindings[parameter] = view.arguments[index]
			}
		}
		for _, recipe := range record.parents {
			if recipe.externalName == "" || recipe.nominalScope != nil || recipe.parameter != nil || recipe.arrayRank != 0 || recipe.wildcard || !validJDK21GenericParentRecipe(recipe, owned, records) {
				return false
			}
			parent := substituteJavaTypeOrigin(recipe, bindings)
			if raw {
				parent.arguments = nil
			}
			parent.declaringOwner = actual.declaringOwner
			if !visit(parent, next) {
				return false
			}
		}
		return true
	}
	if !visit(actual, map[string]bool{}) || !found {
		return inferredJavaTypeOrigin{}, false
	}
	return result, true
}

func completeJDK21GenericOrigin(value inferredJavaTypeOrigin) bool {
	if !completeJavaTypeOrigin(value) || value.arrayRank < 0 {
		return false
	}
	for _, arg := range value.arguments {
		if !completeJDK21GenericOrigin(arg) {
			return false
		}
	}
	return true
}

func validJDK21GenericParentRecipe(value inferredJavaTypeOrigin, owned map[*symbol.TypeParamDeclaration]bool, records map[string]jdk21GenericParentRecord) bool {
	if value.unresolved || value.nominalScope != nil || value.arrayRank < 0 || value.wildcard {
		return false
	}
	if value.parameter != nil {
		return owned[value.parameter] && value.externalName == "" && len(value.arguments) == 0
	}
	record, known := records[value.externalName]
	if value.externalName == "" || !known || len(value.arguments) != 0 && len(value.arguments) != len(record.parameters) {
		return false
	}
	for _, arg := range value.arguments {
		if !validJDK21GenericParentRecipe(arg, owned, records) {
			return false
		}
	}
	return true
}
