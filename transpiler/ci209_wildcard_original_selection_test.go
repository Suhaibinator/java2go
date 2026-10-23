package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

var ci209WildcardOriginalSources = map[string]map[string]string{
	"capture": {
		"library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static <E extends Number> int select(Collection<E> values) { return 1; }
 public static int select(Object value) { return 2; }
 public static <E extends Number> int pair(Collection<E> first, Collection<E> second) { return 1; }
 public static int pair(Object first, Object second) { return 2; }
}
`,
		"probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
public class Main {
 public static void main(String[] args) {
  List<? extends Integer> values = new ArrayList<Integer>();
  System.out.println(select(values));
 }
}
`,
	},
	"lower": {
		"library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static int trace;
 public static <T> int select(Collection<? super T> values) { trace = trace * 10 + 1; return 1; }
 public static int select(Object value) { trace = trace * 10 + 2; return 2; }
}
`,
		"probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
import library.Selectors;
public class Main {
 public static void main(String[] args) {
  List<String> strings = new ArrayList<String>(); strings.add("retained");
  List<Object> objects = new ArrayList<Object>(); objects.add(strings);
  int first = select(strings);
  int second = select(objects);
  System.out.println(first + ":" + second + ":" + Selectors.trace + ":" + strings.get(0) + ":" + (objects.get(0) == strings));
 }
}
`,
	},
}

// The fixture strings above are extracted byte-for-byte from the two original
// hosted-CI parents. No Java declaration, expected output, or compiler path is replaced.
func TestCI209WildcardOriginalDeclarationSelection(t *testing.T) {
	for _, row := range []struct {
		name, fixture, argument, element, parameter, inferred string
		wildcard                                              bool
	}{
		{"upper_capture", "capture", "values", "java.lang.Integer", "E", "java.lang.Integer", true},
		{"lower_strings", "lower", "strings", "java.lang.String", "T", "java.lang.String", false},
		{"lower_objects", "lower", "objects", "java.lang.Object", "T", "java.lang.Object", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			root := sourceGenericViewDemandTestContext(t, ci209WildcardOriginalSources[row.fixture])
			library, callerOwner := findQualifiedSourceClass("library.Selectors"), findQualifiedSourceClass("probe.Main")
			if library == nil || callerOwner == nil {
				t.Fatal("original source class scopes missing")
			}
			var generic, object *symbol.Definition
			for _, method := range library.Methods {
				if method.OriginalName != "select" {
					continue
				}
				if len(method.TypeParameters) == 1 {
					if generic != nil {
						t.Fatal("duplicate original generic overload")
					}
					generic = method
				} else {
					if object != nil {
						t.Fatal("duplicate original Object overload")
					}
					object = method
				}
			}
			if generic == nil || object == nil || !generic.IsStatic || !object.IsStatic || len(generic.Parameters) != 1 || len(object.Parameters) != 1 || object.Parameters[0].OriginalType != "Object" {
				t.Fatal("exact original overloads missing")
			}
			binder := generic.TypeParameters[0]
			if binder.Name != row.parameter || binder.Declaration == nil || definitionDeclaredJavaType(generic.Parameters[0]).TypeParameterBindings[row.parameter] != binder.Declaration {
				t.Fatal("formal must retain exact callee method binder pointer")
			}
			declaring, owned := genericMethodOwnerContext(generic)
			if !owned || declaring.currentClass != library || declaring.currentFile == nil || declaring.currentFile != findFileScopeForClassScope(library) || declaring.currentFile.ParsedRoot() == nil || !bytes.Equal(declaring.currentFile.Source, []byte(ci209WildcardOriginalSources[row.fixture]["library/Selectors.java"])) || library.Class == nil || library.Class.DeclarationNode == nil || !threadSourceNodeWithin(library.Class.DeclarationNode, declaring.currentFile.ParsedRoot()) || generic.DeclarationNode == nil || !threadSourceNodeWithin(generic.DeclarationNode, library.Class.DeclarationNode) || !genericMethodClassHeaderOwned(generic, declaring.currentFile.Source) {
				t.Fatal("registered original callee file/root/method/header ownership missing")
			}
			declaring.localScope = generic
			if visibleTypeParameterDeclarationForJavaType(row.parameter, declaring) != binder.Declaration {
				t.Fatal("actual callee header binder is not lexically visible")
			}
			formalNodes := nodeutil.SemanticNamedChildrenOf(generic.DeclarationNode.ChildByFieldName("parameters"))
			if len(formalNodes) != 1 || formalNodes[0].Type() != "formal_parameter" {
				t.Fatal("original formal TYPE attachment missing")
			}
			formalType, formalName := formalNodes[0].ChildByFieldName("type"), formalNodes[0].ChildByFieldName("name")
			if formalType == nil || formalName == nil || !threadSourceNodeWithin(formalType, generic.DeclarationNode) || !threadSourceNodeWithin(formalName, generic.DeclarationNode) || formalType.Content(declaring.currentFile.Source) != generic.Parameters[0].OriginalType || formalName.Content(declaring.currentFile.Source) != generic.Parameters[0].OriginalName {
				t.Fatal("actual retained formal TYPE/name differs from parsed method")
			}
			if row.fixture == "capture" {
				if len(binder.Bounds) != 1 {
					t.Fatal("original E upper bound missing")
				}
				bound := declaredJavaTypeOrigin(binder.Bounds[0], declaring)
				if bound.parameter != nil || bound.nominalScope != nil || bound.externalName != "java.lang.Number" || bound.unresolved {
					t.Fatal("original E upper bound lost canonical declaration")
				}
			} else if len(binder.Bounds) != 0 {
				t.Fatal("original unbounded T header changed")
			}
			caller := callerOwner.FindMethodByName("main", nil)
			if caller == nil || caller.DeclarationNode == nil {
				t.Fatal("original main declaration missing")
			}
			ctx := classScopeCtx(callerOwner, root)
			ctx.localScope = caller
			ctx.expectedType, ctx.expectedTypeRoot = "", nil
			if ctx.currentFile == nil || ctx.currentFile != findFileScopeForClassScope(callerOwner, ctx) || ctx.currentFile.ParsedRoot() == nil {
				t.Fatal("caller file and parsed root ownership missing")
			}
			source := ctx.currentFile.Source
			var call *sitter.Node
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil {
					return
				}
				if node.Type() == "method_invocation" && node.ChildByFieldName("object") == nil {
					name := node.ChildByFieldName("name")
					args := nodeutil.SemanticNamedChildrenOf(node.ChildByFieldName("arguments"))
					if name != nil && name.Content(source) == "select" && len(args) == 1 && args[0].Content(source) == row.argument {
						if call != nil {
							t.Fatal("duplicate original invocation")
						}
						call = node
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					walk(child)
				}
			}
			walk(caller.DeclarationNode)
			if call == nil {
				t.Fatal("exact original select invocation missing")
			}
			args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
			if len(args) != 1 || args[0].Type() != "identifier" || caller.FindVariable(row.argument) == nil {
				t.Fatal("exact original local argument missing")
			}
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(args[0], ctx, source, &actual)
			actual.javaType = typ
			formals := methodFormalOrigins(generic, library, nil, ctx)
			if !known || actual.unresolved || actual.nominalScope != nil || actual.parameter != nil || actual.arrayRank != 0 || actual.externalName != "java.util.List" || len(actual.arguments) != 1 || actual.arguments[0].externalName != row.element || actual.arguments[0].nominalScope != nil || actual.arguments[0].parameter != nil || actual.arguments[0].arrayRank != 0 || actual.arguments[0].wildcard != row.wildcard {
				t.Fatal("original argument declaration and wildcard origin premise failed")
			}
			if len(formals) != 1 || formals[0].externalName != "java.util.Collection" || formals[0].nominalScope != nil || formals[0].arrayRank != 0 || len(formals[0].arguments) != 1 {
				t.Fatal("canonical original Collection formal premise failed")
			}
			if row.fixture == "capture" && (formals[0].arguments[0].parameter != binder.Declaration || formals[0].arguments[0].wildcard) {
				t.Fatal("exact upper-capture formal binder missing")
			}
			if row.fixture == "lower" && (!formals[0].arguments[0].wildcard || generic.Parameters[0].OriginalType != "Collection<? super T>") {
				t.Fatal("original lower wildcard formal missing")
			}
			constraints, boundsOK := methodInvocationConstraintBindings(generic, library, args, false, ctx, source)
			constraintOrigin := declaredJavaTypeOrigin(symbol.JavaType{Original: constraints[row.parameter], TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
			if !boundsOK || len(constraints) != 1 || constraintOrigin.unresolved || constraintOrigin.nominalScope != nil || constraintOrigin.parameter != nil || constraintOrigin.wildcard || constraintOrigin.arrayRank != 0 || len(constraintOrigin.arguments) != 0 || !sameJavaTypeOrigin(constraintOrigin, intrinsicJavaTypeOrigin(row.inferred)) {
				t.Fatalf("original feasible declaration-owned canonical constraints must complete before score assertion: applicable=%v bindings=%v origin=%+v", boundsOK, constraints, constraintOrigin)
			}
			projected, proved := declaredMethodArgumentInferredOrigin(formals[0], generic, formals, args, false, ctx, source)
			projectedCost, projectedExact, projectedApplies, projectedHandled := declaredOriginConversion(actual, projected, ctx)
			initialCost, initialExact, initialApplies, initialHandled := declaredOriginConversion(actual, formals[0], ctx)
			expected := receiverInstantiatedMethodFormal(generic.Parameters[0], generic, library, nil, ctx)
			names := methodCandidateTypeParameterNames(library, generic)
			fallbackCost, fallbackExact, fallbackApplies := javaInvocationConversionCost(args[0], expected, names, ctx, source)
			genericScore, genericOK := scoreMethodCandidate(generic, library, names, args, ctx, source)
			objectScore, objectOK := scoreMethodCandidate(object, library, methodCandidateTypeParameterNames(library, object), args, ctx, source)
			resolved := resolveStaticImportedMethod(call, ctx, source)
			if !genericOK || !objectOK || resolved.source == nil || resolved.source.owner != library || resolved.source.def != generic && resolved.source.def != object {
				t.Fatal("both original candidates must be applicable and selected target must be one exact source declaration")
			}
			t.Logf("WILDCARD209_FACT case=%s known=%v calleeBinderOwned=true actual=%q formal=%q bounds=%v bindings=%v initial=%d,%v,%v,%v fallback=%d,%v,%v projected=%v projectedOwner=%q projectedConversion=%d,%v,%v,%v generic=%+v object=%+v selectedGeneric=%v selectedObject=%v", row.name, known, typ, expected, boundsOK, constraints, initialCost, initialExact, initialApplies, initialHandled, fallbackCost, fallbackExact, fallbackApplies, proved, projected.externalName, projectedCost, projectedExact, projectedApplies, projectedHandled, genericScore, objectScore, resolved.source.def == generic, resolved.source.def == object)
			if !methodCandidateScoreBetter(genericScore, objectScore) || resolved.source.def != generic {
				t.Fatalf("WILDCARD209_DISPATCH: %s applicable generic Collection overload must beat exact Object fallback", row.name)
			}
		})
	}
}
