package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Java's raw first-bound erasure is source metadata, not necessarily a legal
// argument to the preserved Go constraint. For T extends List<String>, Java
// erases T to List, while the Go declaration requires *List[*JavaString]. Keep
// that source view unchanged and choose its constrained Go argument separately.
func sourceClassGoTypeArgumentExprs(
	scope *symbol.ClassScope,
	sourceArguments []string,
	receiver *symbol.ClassScope,
	receiverArguments []string,
	typeParameters []string,
	ctx Ctx,
) []ast.Expr {
	arguments := normalizeClassTypeArguments(scope, sourceArguments, receiver, receiverArguments)
	raw := sourceClassRawArgumentSlots(scope, sourceArguments, receiver, receiverArguments)
	var lexicalSlots []ast.Expr
	if scope != nil && receiver != nil && receiverArguments == nil && ctx.currentClass != nil && (ctx.localScope == nil || !ctx.localScope.IsStatic) {
		hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
		providedHidden := len(sourceArguments) - len(scope.OwnTypeParameters())
		if providedHidden < 0 {
			providedHidden = 0
		}
		// Written arguments still use Java's method/local lexical context. Only a
		// synthesized enclosing-instance slot already identifies its class binder;
		// parsing its Go alias as Java text would capture a shadowing method binder.
		for index := providedHidden; index < hidden && index < len(arguments); index++ {
			declaration := scope.TypeParameters[index].Declaration
			if declaration == nil || strings.TrimSpace(declaration.GoName) == "" {
				continue
			}
			for _, parameter := range receiver.TypeParameters {
				if parameter.Declaration != declaration {
					continue
				}
				for _, lexical := range ctx.currentClass.TypeParameters {
					if lexical.Declaration == declaration {
						if lexicalSlots == nil {
							lexicalSlots = make([]ast.Expr, len(arguments))
						}
						lexicalSlots[index] = ast.NewIdent(declaration.GoName)
						break
					}
				}
				break
			}
		}
	}
	return sourceClassGoArgumentsWithLexicalSlots(scope, arguments, raw, typeParameters, ctx, lexicalSlots)
}

func sourceClassRawArgumentSlots(scope *symbol.ClassScope, sourceArguments []string, receiver *symbol.ClassScope, receiverArguments []string) []bool {
	if scope == nil {
		return nil
	}
	raw := make([]bool, len(scope.TypeParameters))
	if len(sourceArguments) == len(raw) {
		return raw
	}
	own := len(scope.OwnTypeParameters())
	hidden := len(raw) - own
	providedHidden := len(sourceArguments) - own
	if providedHidden < 0 {
		providedHidden = 0
	}
	if providedHidden > hidden {
		providedHidden = hidden
	}
	available := receiverClassTypeArgumentBindings(receiver, receiverArguments)
	for index := 0; index < hidden; index++ {
		argument, found := available.argumentFor(scope.TypeParameters[index])
		found = found && strings.TrimSpace(argument) != ""
		raw[index] = index >= providedHidden && !found
	}
	for index := hidden; index < len(raw); index++ {
		raw[index] = providedHidden+index-hidden >= len(sourceArguments)
	}
	return raw
}

func sourceClassGoArguments(scope *symbol.ClassScope, arguments []string, raw []bool, typeParameters []string, ctx Ctx) []ast.Expr {
	return sourceClassGoArgumentsWithLexicalSlots(scope, arguments, raw, typeParameters, ctx, nil)
}

func sourceClassGoArgumentsWithLexicalSlots(scope *symbol.ClassScope, arguments []string, raw []bool, typeParameters []string, ctx Ctx, lexicalSlots []ast.Expr) []ast.Expr {
	result := make([]ast.Expr, len(arguments))
	// Seed before dependent bounds substitute binder slots. The Java argument
	// strings and every unseeded raw/wildcard representation remain unchanged.
	copy(result, lexicalSlots)
	if scope == nil {
		for index, argument := range arguments {
			if result[index] == nil {
				result[index] = javaTypeStringToGoTypeExpr(argument, typeParameters, ctx)
			}
		}
		return result
	}
	needsBound := false
	for index, argument := range arguments {
		if index < len(scope.TypeParameters) && sourceClassArgumentNeedsBound(argument, index < len(raw) && raw[index]) {
			needsBound = true
			break
		}
	}
	if !needsBound {
		for index, argument := range arguments {
			if result[index] == nil {
				result[index] = javaTypeStringToGoTypeExpr(argument, typeParameters, ctx)
			}
		}
		return result
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	parameters := qualifyTypeParameterBounds(scope.TypeParameters, declaring)
	lookup := newTypeParameterLookup(parameters)
	binderSlots := make(map[string]int, len(parameters))
	for index, parameter := range parameters {
		binderSlots[parameter.EmittedName()] = index
	}
	// A bound is interpreted in its declaration namespace, then its Go nominal
	// names are qualified for the consuming file. Caller String/List/member
	// declarations therefore cannot capture a declaration's bound.
	var nominals map[string]*symbol.ClassScope
	nominalFor := func(name string) *symbol.ClassScope {
		if name == "any" {
			return nil
		}
		if nominals == nil {
			nominals = make(map[string]*symbol.ClassScope)
			for _, candidate := range allSourceClassScopes() {
				if candidate.Class != nil && findJavaPackageForClassScope(candidate) == findJavaPackageForClassScope(scope) {
					nominals[candidate.Class.Name] = candidate
				}
			}
		}
		return nominals[name]
	}
	building := make([]bool, len(arguments))
	var argumentExpr func(int) ast.Expr
	var bind func(ast.Expr) ast.Expr
	bind = func(expr ast.Expr) ast.Expr {
		switch value := expr.(type) {
		case *ast.Ident:
			if index, found := binderSlots[value.Name]; found && index < len(arguments) {
				return argumentExpr(index)
			}
			if nominal := nominalFor(value.Name); nominal != nil {
				return qualifiedNameExpr(nominal.Class.Name, findJavaPackageForClassScope(nominal), ctx)
			}
			return value
		case *ast.StarExpr:
			return &ast.StarExpr{X: bind(value.X)}
		case *ast.IndexExpr:
			return &ast.IndexExpr{X: bind(value.X), Index: bind(value.Index)}
		case *ast.IndexListExpr:
			indices := make([]ast.Expr, len(value.Indices))
			for index, argument := range value.Indices {
				indices[index] = bind(argument)
			}
			return &ast.IndexListExpr{X: bind(value.X), Indices: indices}
		case *ast.ArrayType:
			return &ast.ArrayType{Len: value.Len, Elt: bind(value.Elt)}
		case *ast.SelectorExpr:
			return value
		default:
			return value
		}
	}
	argumentExpr = func(index int) ast.Expr {
		if result[index] != nil {
			return result[index]
		}
		// A recursive binder in an invariant pointer bound need not have a finite
		// Go solution. Bound recursion at the existing wildcard fallback; the
		// declaration constraints and complete-family admission checks stay intact.
		if building[index] {
			return &ast.Ident{Name: "any"}
		}
		building[index] = true
		defer func() { building[index] = false }()
		argument := arguments[index]
		if index >= len(parameters) || !sourceClassArgumentNeedsBound(argument, index < len(raw) && raw[index]) {
			result[index] = javaTypeStringToGoTypeExpr(argument, typeParameters, ctx)
			return result[index]
		}
		bounds := goRepresentableTypeParameterBounds(parameters[index].Bounds, lookup, nil)
		// Java erasure selects the first intersection bound. A concrete-pointer
		// Go constraint cannot itself be used as a value type; retain that first
		// bound as the argument and leave every declared constraint intact.
		if len(bounds) > 1 {
			bounds = bounds[:1]
		}
		result[index] = bind(constraintExprInContext(bounds, scope.GoTypeParameterNames(), declaring))
		return result[index]
	}
	for index := range arguments {
		argumentExpr(index)
	}
	return result
}

func sourceClassArgumentNeedsBound(argument string, raw bool) bool {
	argument = strings.TrimSpace(argument)
	if raw || argument == "?" {
		return true
	}
	if !strings.HasPrefix(argument, "?") {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(argument, "?")), "super")
}

// Ancestor walkers already have Go expressions for the child's arguments.
// Replace only binder identifiers, never package names or selector spellings.
func substituteSourceClassGoArgument(expr ast.Expr, bindings map[string]ast.Expr) ast.Expr {
	switch value := expr.(type) {
	case *ast.Ident:
		if replacement := bindings[value.Name]; replacement != nil {
			return replacement
		}
		return value
	case *ast.StarExpr:
		return &ast.StarExpr{X: substituteSourceClassGoArgument(value.X, bindings)}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: substituteSourceClassGoArgument(value.X, bindings), Index: substituteSourceClassGoArgument(value.Index, bindings)}
	case *ast.IndexListExpr:
		indices := make([]ast.Expr, len(value.Indices))
		for index, argument := range value.Indices {
			indices[index] = substituteSourceClassGoArgument(argument, bindings)
		}
		return &ast.IndexListExpr{X: substituteSourceClassGoArgument(value.X, bindings), Indices: indices}
	case *ast.ArrayType:
		return &ast.ArrayType{Len: value.Len, Elt: substituteSourceClassGoArgument(value.Elt, bindings)}
	case *ast.SelectorExpr:
		return value
	default:
		return value
	}
}
