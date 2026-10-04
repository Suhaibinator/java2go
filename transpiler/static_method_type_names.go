package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Static Java methods become package-level Go functions, sharing Go's type
// namespace. Source class identities stay unchanged; rename the resolved method
// so declarations, invocations and method references all use the same symbol.
func packageHasEmittedSourceTypeName(pkg *symbol.PackageScope, name string) bool {
	if pkg == nil {
		return false
	}
	for _, file := range pkg.Files {
		for _, top := range file.TopLevelClasses {
			if visitClassScopes(top, func(scope *symbol.ClassScope) bool {
				return scope.Class != nil && scope.Class.Name == name
			}) {
				return true
			}
		}
	}
	return false
}
