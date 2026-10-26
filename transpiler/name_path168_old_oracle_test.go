package transpiler

import "github.com/NickyBoy89/java2go/symbol"

func namePath168OldGeneratedIdentifierExists(name string, owner *symbol.ClassScope) bool {
	matchesScope := func(scope *symbol.ClassScope) bool {
		if scope == nil {
			return false
		}
		if scope.Class != nil && scope.Class.Name == name {
			return true
		}
		for _, field := range scope.Fields {
			if field != nil && field.Name == name {
				return true
			}
		}
		for _, method := range scope.Methods {
			if method != nil && method.Name == name {
				return true
			}
		}
		for _, enumConstant := range scope.EnumConstants {
			if enumConstant.EmittedName() == name {
				return true
			}
		}
		return name == classDispatchTypeName(scope) ||
			name == classDispatchFieldName(scope) ||
			name == classSelfSetterName(scope) ||
			name == interfaceDefaultCarrierName(scope)
	}
	if matchesScope(owner) {
		return true
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			for _, top := range file.TopLevelClasses {
				if visitClassScopes(top, matchesScope) {
					return true
				}
			}
		}
	}
	return false
}
