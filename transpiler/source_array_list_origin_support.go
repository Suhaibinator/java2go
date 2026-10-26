package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Pure declaration-origin/render helpers only; no anonymous, diamond or sealed ABI admission.
func sourceDiamondOriginEdge(origin inferredJavaTypeOrigin) (string, map[string]*symbol.TypeParamDeclaration, bool) {
	bindings := map[string]*symbol.TypeParamDeclaration{}
	var render func(inferredJavaTypeOrigin) (string, bool)
	render = func(value inferredJavaTypeOrigin) (string, bool) {
		if !completeJavaTypeOrigin(value) || value.wildcard {
			return "", false
		}
		name := ""
		switch {
		case value.parameter != nil:
			name = value.parameter.SourceName
			if old := bindings[name]; old != nil && old != value.parameter {
				return "", false
			}
			if name == "" {
				return "", false
			}
			bindings[name] = value.parameter
		case value.nominalScope != nil:
			name = qualifiedSourceClassName(value.nominalScope)
		case value.externalName != "":
			name = value.externalName
		default:
			return "", false
		}
		if name == "" {
			return "", false
		}
		if len(value.arguments) > 0 {
			args := make([]string, len(value.arguments))
			for i, arg := range value.arguments {
				s, ok := render(arg)
				if !ok {
					return "", false
				}
				args[i] = s
			}
			name += "<" + strings.Join(args, ", ") + ">"
		}
		return name + strings.Repeat("[]", value.arrayRank), true
	}
	edge, ok := render(origin)
	return edge, bindings, ok
}

func sourceNativeFunctionalResultOrigin(scope *symbol.ClassScope, result string, declaring Ctx) inferredJavaTypeOrigin {
	bindings, owned := sourceNativeFunctionalResultBindings(scope, declaring)
	if !owned {
		return inferredJavaTypeOrigin{javaType: result, unresolved: true}
	}
	return declaredJavaTypeOrigin(symbol.JavaType{Original: result, TypeParameterBindings: bindings}, declaring)
}

func sourceNativeFunctionalResultBindings(scope *symbol.ClassScope, declaring Ctx) (map[string]*symbol.TypeParamDeclaration, bool) {
	if _, owned := supplierSourceDeclarationCtx(scope, declaring); !owned {
		return nil, false
	}
	bindings := map[string]*symbol.TypeParamDeclaration{}
	for _, parameter := range scope.TypeParameters {
		alias := parameter.EmittedName()
		if parameter.Declaration == nil || alias == "" {
			return nil, false
		}
		if _, duplicate := bindings[alias]; duplicate {
			return nil, false
		}
		bindings[alias] = parameter.Declaration
	}
	return bindings, true
}

func parseDeferredCallable(argument *sitter.Node, target string, parameters []string, result string, ctx Ctx, source []byte) ast.Expr {
	if argument == nil {
		return nil
	}
	argumentCtx := intrinsicLambdaParseContext(ctx, argument, lambdaArgumentTypes{paramJavaTypes: parameters, resultJavaType: result})
	argumentCtx.expectedType = target
	argumentCtx.expectedTypeRoot = argument
	return ParseExpr(argument, source, argumentCtx)
}

func deferredToCollectionFactoryName(collector *sitter.Node, source []byte, contexts ...Ctx) (string, bool) {
	if collector == nil || collector.Type() != "method_invocation" || len(contexts) != 1 {
		return "", false
	}
	ctx := contexts[0]
	if !sourceArrayListNativeSourceContext(collector, ctx, source) || !sourceArrayListConstructorFileNamespace(ctx.currentFile) {
		return "", false
	}
	object := collector.ChildByFieldName("object")
	name := collector.ChildByFieldName("name")
	if object == nil || name == nil {
		return "", false
	}
	selected, known := intrinsicStaticClassName(object, ctx, source)
	if !known || selected != "Collectors" || sourceGenericViewLocalTypeShadowed(object, object.Content(source), source) {
		return "", false
	}
	// The original TYPE namespace, rather than a migrated intrinsic registry,
	// supplies this declaration identity. Conflicting explicit imports cannot
	// acquire authority from the last entry retained in a mutable import map.
	if object.Type() == "identifier" {
		importedOwner := ""
		root := ctx.currentFile.ParsedRoot()
		for index := 0; index < int(root.NamedChildCount()); index++ {
			declaration := root.NamedChild(index)
			if declaration.Type() != "import_declaration" {
				continue
			}
			imported, ok := nodeutil.ReadJavaImportSyntax(declaration, source)
			if !ok {
				return "", false
			}
			if imported.Kind != nodeutil.JavaImportSingleType || imported.Name != object.Content(source) {
				continue
			}
			owner := imported.Scope + "." + imported.Name
			if importedOwner != "" && importedOwner != owner {
				return "", false
			}
			importedOwner = owner
		}
	}
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	owner := declaredJavaTypeOrigin(symbol.JavaType{Original: object.Content(source), TypeParameterBindings: bindings}, ctx)
	if owner.unresolved || owner.wildcard || owner.arrayRank != 0 || owner.parameter != nil || owner.nominalScope != nil || owner.externalName != "java.util.stream.Collectors" || len(owner.arguments) != 0 {
		return "", false
	}
	return name.Content(source), true
}
