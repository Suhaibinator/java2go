package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
)

// Source text metadata names only the execution entry for an actual Java
// toString declaration (or Enum's generated default). Ordinary Java methods
// with Go formatting or former protocol spellings retain their own selectors.
func sourceToStringRegistration(scope *symbol.ClassScope, id string, ctx Ctx) ast.Stmt {
	if scope == nil || scope.Class == nil || scope.IsInterface {
		return nil
	}
	ownerCtx := classScopeCtx(scope, ctx)
	selector := ""
	if method := findDeclaredToStringMethod(scope); method != nil && method.HasBody {
		selector = executionImplementationName(method, scope, ownerCtx)
	} else if scope.IsEnum {
		selector = executionStringMethodName(scope)
	}
	if selector == "" {
		return nil
	}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterJavaSourceToString", javaTypeIDLiteral(id, ctx), &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(selector)})}
}

func syntheticSourceTextScope(structName string, ctx Ctx) *symbol.ClassScope {
	if ctx.currentClass != nil && ctx.currentClass.Class != nil && ctx.currentClass.Class.Name == structName {
		return ctx.currentClass
	}
	for _, local := range ctx.localClasses {
		if local != nil && local.structName == structName {
			return local.scope
		}
	}
	for _, anonymous := range ctx.anonymousClasses {
		if anonymous != nil && anonymous.structName == structName {
			return anonymous.scope
		}
	}
	return nil
}

// Native fmt.Stringer is optional: it cannot occupy the same Go selector as a
// real source member or an embedded source type. Java conversion uses the exact
// registered execution entry regardless of whether this convenience exists.
func sourceOwnsNativeStringSelector(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	queue := []*symbol.ClassScope{scope}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] {
			continue
		}
		seen[current] = true
		for _, field := range current.Fields {
			if field != nil && !field.IsStatic && field.Name == "String" {
				return true
			}
		}
		for _, method := range current.Methods {
			if method != nil && !method.IsStatic && !method.Constructor && method.Name == "String" {
				return true
			}
		}
		parents := append([]*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(ctx, current)}, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		if current == scope {
			for _, embedded := range parents {
				if embedded != nil && embedded.Class != nil && embedded.Class.Name == "String" {
					return true
				}
			}
		}
		queue = append(queue, parents...)
	}
	return false
}
