package transpiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"

	sitter "github.com/smacker/go-tree-sitter"
)

// A direct anonymous member selection retains the anonymous declaration type,
// which exists only after hoisting its receiver. Prepare that exact source root
// before field admission, and reuse its emitted receiver without constructing
// or hoisting it a second time. The cache is lexical to this expression's Ctx.
func prepareAnonymousFieldReceiver(node *sitter.Node, source []byte, ctx Ctx) Ctx {
	target := node
	for target != nil && (target.Type() == "parenthesized_expression" || target.Type() == "expression_statement") && target.NamedChildCount() == 1 {
		target = target.NamedChild(0)
	}
	if target == nil {
		return ctx
	}
	switch target.Type() {
	case "assignment_expression":
		assignment := target
		target = assignment.ChildByFieldName("left")
		if target == nil {
			target = assignment.Child(0)
		}
	case "update_expression":
		if target.NamedChildCount() != 1 {
			return ctx
		}
		target = target.NamedChild(0)
	}
	for target != nil && target.Type() == "parenthesized_expression" && target.NamedChildCount() == 1 {
		target = target.NamedChild(0)
	}
	if target == nil || target.Type() != "field_access" {
		return ctx
	}
	receiver := target.ChildByFieldName("object")
	if anonymousCreationExpressionRoot(receiver) == nil {
		return ctx
	}
	if _, prepared := preparedAnonymousFieldReceiverExpr(receiver, source, ctx); prepared {
		return ctx
	}
	expression := ParseExpr(receiver, source, ctx)
	prepared := ctx
	prepared.preparedAnonymousReceiverRoot = receiver
	prepared.preparedAnonymousReceiverExpr = expression
	prepared.preparedAnonymousReceiverStamp = anonymousReceiverStamp(source, prepared)
	return prepared
}

func preparedAnonymousFieldReceiverExpr(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	root := ctx.preparedAnonymousReceiverRoot
	if node == nil || root == nil || ctx.preparedAnonymousReceiverExpr == nil {
		return nil, false
	}
	if node.StartByte() != root.StartByte() || node.EndByte() != root.EndByte() || node.Type() != root.Type() {
		return nil, false
	}
	if ctx.preparedAnonymousReceiverStamp != anonymousReceiverStamp(source, ctx) {
		return nil, false
	}
	return ctx.preparedAnonymousReceiverExpr, true
}

// The memo is valid only in the same file/emission, lexical declaration and
// physical ABI. Source offsets alone cannot distinguish an AST containing the
// previous caller's Execution, captured local names or generic substitutions.
type anonymousReceiverEmissionStamp struct {
	source                                                  *byte
	sourceLength                                            int
	file                                                    *symbol.FileScope
	class                                                   *symbol.ClassScope
	local, erased                                           *symbol.Definition
	lexicalBody, expectedRoot                               *sitter.Node
	hoisted                                                 *[]ast.Decl
	counter                                                 *int
	families                                                *genericFamilyAnalysis
	witnesses                                               *dependentTypeWitnessPlan
	execution, className, expectedType, physicalABI, locals string
}

func anonymousReceiverStamp(source []byte, ctx Ctx) anonymousReceiverEmissionStamp {
	return anonymousReceiverEmissionStamp{source: anonymousReceiverSourceIdentity(source), sourceLength: len(source), file: ctx.currentFile, class: ctx.currentClass, local: ctx.localScope, erased: ctx.erasedGenericMethodBody, lexicalBody: ctx.localBindingBody, expectedRoot: ctx.expectedTypeRoot, hoisted: ctx.hoistedDecls, counter: ctx.anonClassCounter, families: ctx.genericFamilies, witnesses: ctx.dependentTypeWitnesses, execution: ctx.executionContextName, className: ctx.className, expectedType: ctx.expectedType, physicalABI: anonymousReceiverPhysicalABI(ctx), locals: anonymousReceiverLocalBindings(ctx.localScope)}
}

func anonymousReceiverPhysicalABI(ctx Ctx) string {
	var out strings.Builder
	fmt.Fprintf(&out, "%q|%q|%q|", inScopeTypeParameters(ctx), ctx.lambdaParameterJavaTypes, ctx.lambdaResultJavaType)
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		fmt.Fprintf(&out, "%q:%q:%p|", parameter.Name, parameter.EmittedName(), parameter.Declaration)
		for _, bound := range parameter.Bounds {
			fmt.Fprintf(&out, "%q:%s|", bound.Original, anonymousReceiverTypeBindings(bound.TypeParameterBindings))
		}
	}
	keys := make([]string, 0, len(ctx.rawGenericParameterTypes))
	for key := range ctx.rawGenericParameterTypes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&out, "%q=%q|", key, ctx.rawGenericParameterTypes[key])
	}
	// Intrinsic type arguments belong to the native generator callback after
	// Java arguments have already been emitted. They cannot change this receiver.
	expressions := []ast.Expr{ctx.lastType}
	for _, expression := range expressions {
		if expression == nil {
			out.WriteString("nil|")
			continue
		}
		var printed bytes.Buffer
		if err := format.Node(&printed, token.NewFileSet(), expression); err != nil {
			fmt.Fprintf(&out, "invalid:%p|", expression)
		} else {
			out.Write(printed.Bytes())
			out.WriteByte('|')
		}
	}
	return out.String()
}

func anonymousReceiverTypeBindings(bindings map[string]*symbol.TypeParamDeclaration) string {
	var out strings.Builder
	keys := make([]string, 0, len(bindings))
	for key := range bindings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		declaration := bindings[key]
		if declaration == nil {
			fmt.Fprintf(&out, "%q=nil|", key)
		} else {
			fmt.Fprintf(&out, "%q:%p:%q:%q|", key, declaration, declaration.SourceName, declaration.GoName)
		}
	}
	return out.String()
}

func anonymousReceiverLocalBindings(scope *symbol.Definition) string {
	var out strings.Builder
	seen := map[*symbol.Definition]bool{}
	var walk func(*symbol.Definition)
	walk = func(definition *symbol.Definition) {
		if definition == nil || seen[definition] {
			return
		}
		seen[definition] = true
		fmt.Fprintf(&out, "%p:%q:%q:%q:%q:%t:%t:%p:%s|", definition, definition.OriginalName, definition.Name, definition.OriginalType, definition.Type, definition.Nullable, definition.IsStatic, definition.DirectTypeParameter, anonymousReceiverTypeBindings(definition.TypeParameterBindings))
		for _, parameter := range definition.Parameters {
			walk(parameter)
		}
		for _, child := range definition.Children {
			walk(child)
		}
	}
	walk(scope)
	return out.String()
}

// SourceFile supplies one immutable byte snapshot while its AST is emitted.
// Its backing identity/length plus currentFile distinguish another file or
// snapshot without rescanning the whole Java source for every memo probe.
func anonymousReceiverSourceIdentity(source []byte) *byte {
	if len(source) == 0 {
		return nil
	}
	return &source[0]
}
