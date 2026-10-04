package transpiler

import (
	"sort"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
)

// Enum constants are implicit fields for allocation and lookup, but remain
// outside Fields: ordinary field codegen must not duplicate their storage or
// run their constructor arguments as ordinary static initializers.
func classFieldBindings(scope *symbol.ClassScope) []*symbol.Definition {
	if scope == nil {
		return nil
	}
	fields := append([]*symbol.Definition(nil), scope.Fields...)
	for _, constant := range scope.EnumConstants {
		if constant.Field != nil {
			fields = append(fields, constant.Field)
		}
	}
	return fields
}

// Static Java fields share Go package scope with fields of every other class,
// static functions, constructors and source/synthetic types. Allocate the
// backing Definition once, before any file emits references or metadata.
// Stable owner/declaration order makes the result independent of file order.
func resolvePackageStaticFieldNames(pkg *symbol.PackageScope) {
	if pkg == nil {
		return
	}
	type binding struct {
		field *symbol.Definition
		key   string
	}
	var bindings []binding
	fixed := map[string]bool{}
	sourceNames := map[string]bool{}
	reserve := func(name string) {
		if name != "" {
			fixed[symbol.GoIdentifier(name)] = true
		}
	}
	for name := range goReservedFuncNames {
		reserve(name)
	}
	for _, name := range goPredeclaredBindingNames() {
		reserve(name)
	}
	for _, file := range pkg.Files {
		if file == nil {
			continue
		}
		for _, top := range file.TopLevelClasses {
			visitClassScopes(top, func(scope *symbol.ClassScope) bool {
				if scope.Class == nil {
					return false
				}
				reserve(scope.Class.Name)
				reserve(classDispatchTypeName(scope))
				reserve(interfaceDefaultCarrierName(scope))
				constructor := defaultConstructorName(scope.Class.Name)
				if !scope.IsInterface && !classHasExplicitConstructor(scope) {
					reserve(constructor)
					if !scope.IsEnum {
						reserve(constructorWithSelfName(constructor))
					}
				}
				for _, method := range scope.Methods {
					if method != nil && (method.IsStatic || method.Constructor) {
						reserve(method.Name)
						if method.Constructor {
							reserve(constructorWithSelfName(method.Name))
						}
					}
				}
				// These enum ABI declarations have fixed names. Execution, storage and
				// generic helpers allocate later through their collision-aware planners,
				// which also observe the implicit fields' allocated names.
				if scope.IsEnum {
					reserve(scope.Class.Name + "Values")
					reserve(scope.Class.Name + "ValueOf")
					reserve("_" + symbol.Lowercase(scope.Class.Name) + "Values")
					for _, constant := range scope.EnumConstants {
						reserve("_" + symbol.Lowercase(scope.Class.Name) + "_ordinal_" + constant.Name)
					}
				}
				for _, field := range classFieldBindings(scope) {
					if field != nil && field.IsStatic {
						bindings = append(bindings, binding{field, javaClassBinaryName(scope)})
						sourceNames[symbol.GoIdentifier(field.Name)] = true
					}
				}
				return false
			})
		}
	}
	sort.SliceStable(bindings, func(i, j int) bool {
		a, b := bindings[i], bindings[j]
		if a.key != b.key {
			return a.key < b.key
		}
		if a.field.DeclarationNode != nil && b.field.DeclarationNode != nil && a.field.DeclarationNode.StartByte() != b.field.DeclarationNode.StartByte() {
			return a.field.DeclarationNode.StartByte() < b.field.DeclarationNode.StartByte()
		}
		return a.field.OriginalName < b.field.OriginalName
	})
	occupied := fixed
	for _, binding := range bindings {
		field := binding.field
		if !symbol.IsReserved(field.Name) && !occupied[symbol.GoIdentifier(field.Name)] {
			occupied[symbol.GoIdentifier(field.Name)] = true
			continue
		}
		base := field.Name
		for suffix := 0; ; suffix++ {
			candidate := base + strconv.Itoa(suffix)
			emitted := symbol.GoIdentifier(candidate)
			if !symbol.IsReserved(candidate) && !occupied[emitted] && !sourceNames[emitted] {
				field.Rename(candidate)
				occupied[emitted] = true
				break
			}
		}
	}
}
