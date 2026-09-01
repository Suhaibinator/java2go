package transpiler

import (
	"fmt"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// TypeInformation captures simplified type metadata extracted from a source file.
type TypeInformation struct {
	Classes map[string]ClassTypeInformation
}

// ClassTypeInformation stores fields and methods for a class, keyed by original Java names.
type ClassTypeInformation struct {
	Fields  map[string]string
	Methods map[string]MethodTypeInformation
}

// MethodTypeInformation stores method return, parameter, and local variable types.
type MethodTypeInformation struct {
	ReturnType string
	Parameters map[string]string
	Locals     map[string]string
}

// ExtractTypeInformation returns type information for a parsed source file.
func ExtractTypeInformation(file parsing.SourceFile) (*TypeInformation, error) {
	if file.Symbols == nil {
		return nil, fmt.Errorf("source file has no symbols")
	}

	info := &TypeInformation{
		Classes: map[string]ClassTypeInformation{},
	}

	var walk func(scope *symbol.ClassScope)
	walk = func(scope *symbol.ClassScope) {
		if scope == nil || scope.Class == nil {
			return
		}

		classInfo := ClassTypeInformation{
			Fields:  map[string]string{},
			Methods: map[string]MethodTypeInformation{},
		}

		for _, field := range scope.Fields {
			if field == nil {
				continue
			}
			classInfo.Fields[field.OriginalName] = field.OriginalType
		}

		for _, method := range scope.Methods {
			if method == nil {
				continue
			}

			methodInfo := MethodTypeInformation{
				ReturnType: method.OriginalType,
				Parameters: map[string]string{},
				Locals:     map[string]string{},
			}

			for _, param := range method.Parameters {
				if param == nil {
					continue
				}
				methodInfo.Parameters[param.OriginalName] = param.OriginalType
			}

			for _, local := range method.Children {
				if local == nil {
					continue
				}
				methodInfo.Locals[local.OriginalName] = local.OriginalType
			}

			classInfo.Methods[method.OriginalName] = methodInfo
		}

		info.Classes[scope.Class.OriginalName] = classInfo

		for _, sub := range scope.Subclasses {
			walk(sub)
		}
	}

	for _, top := range file.Symbols.TopLevelClasses {
		walk(top)
	}

	return info, nil
}
