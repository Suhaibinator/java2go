package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
	"testing"
)

type wildcardStrictScoreFixture struct {
	ctx    Ctx
	owner  *symbol.ClassScope
	method *symbol.Definition
	caller *symbol.Definition
	call   *sitter.Node
	args   []*sitter.Node
	source []byte
}

// These are parser/scorer shields, including intentionally invalid Java calls
// and damaged metadata. They are not replacement application or JVM oracles.
func wildcardStrictScoreSetup(t *testing.T, header, formal, locals, arguments string, extra map[string]string) *wildcardStrictScoreFixture {
	t.Helper()
	target := "library.Selectors.select"
	if strings.HasPrefix(arguments, "<Integer>:") {
		target = "library.Selectors.<Integer>select"
		arguments = strings.TrimPrefix(arguments, "<Integer>:")
	}
	sources := map[string]string{
		"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;public class Selectors{public static ` + header + ` int select(` + formal + `){return 1;}public static int select(Object value){return 2;}}`,
		"probe/Main.java":        `package probe;import java.util.List;import java.util.Collection;import java.lang.Integer;import java.lang.Number;public class Main{public static <Q extends Number> int run(){` + locals + `return ` + target + `(` + arguments + `);}}`,
	}
	for path, source := range extra {
		sources[path] = source
	}
	root := sourceGenericViewDemandTestContext(t, sources)
	owner, callerOwner := findQualifiedSourceClass("library.Selectors"), findQualifiedSourceClass("probe.Main")
	if owner == nil || callerOwner == nil {
		t.Fatal("registered source owners absent")
	}
	var method *symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate.OriginalName == "select" && len(candidate.TypeParameters) > 0 {
			if method != nil {
				t.Fatal("duplicate generic candidate")
			}
			method = candidate
		}
	}
	caller := callerOwner.FindMethodByName("run", nil)
	if method == nil || caller == nil || method.DeclarationNode == nil || caller.DeclarationNode == nil {
		t.Fatal("registered source methods absent")
	}
	ctx := classScopeCtx(callerOwner, root)
	ctx.localScope = caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	if ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil || !threadSourceNodeWithin(caller.DeclarationNode, ctx.currentFile.ParsedRoot()) {
		t.Fatal("actual caller file/tree premise absent")
	}
	declaring, owned := genericMethodOwnerContext(method)
	if !owned || declaring.currentClass != owner || declaring.currentFile == nil || !threadSourceNodeWithin(method.DeclarationNode, declaring.currentFile.ParsedRoot()) || !genericMethodClassHeaderOwned(method, declaring.currentFile.Source) {
		t.Fatal("actual callee header/file premise absent")
	}
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			if name := node.ChildByFieldName("name"); name != nil && name.Content(ctx.currentFile.Source) == "select" {
				if call != nil {
					t.Fatal("duplicate scored invocation")
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
		t.Fatal("actual invocation absent")
	}
	return &wildcardStrictScoreFixture{ctx: ctx, owner: owner, method: method, caller: caller, call: call, args: nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments")), source: ctx.currentFile.Source}
}

func wildcardStrictScoreForeign(t *testing.T, source []byte, owner string) *symbol.ClassScope {
	t.Helper()
	file := parsing.SourceFile{Name: "foreign.java", Source: append([]byte(nil), source...)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	scope := file.Symbols.FindClassScope(owner)
	if scope == nil {
		t.Fatal("independent foreign tree absent")
	}
	return scope
}

func wildcardStrictScoreFreshInputs(t *testing.T, f *wildcardStrictScoreFixture) {
	t.Helper()
	declaring, owned := genericMethodOwnerContext(f.method)
	if !owned || declaring.currentClass != f.owner || declaring.currentFile == nil || f.ctx.currentFile == nil {
		t.Fatal("registered independent input owners absent")
	}
	for _, file := range []*symbol.FileScope{f.ctx.currentFile, declaring.currentFile} {
		input := file.SourceInputSnapshot()
		if input == nil || input.Validate() != nil || file.ParsedRoot() == nil {
			t.Fatal("actual ParseSymbols input attachment is not initially current")
		}
	}
}

func wildcardStrictScoreOwnedBeforeMutation(t *testing.T, f *wildcardStrictScoreFixture) methodCandidateScore {
	t.Helper()
	wildcardStrictScoreFreshInputs(t, f)
	if len(f.args) != 1 || f.args[0].Type() != "identifier" || len(f.method.Parameters) != 1 || len(f.method.TypeParameters) != 1 {
		t.Fatal("unmutated authority fixture arity/binder premise absent")
	}
	var actual inferredJavaTypeOrigin
	typ, known := inferExprJavaType(f.args[0], f.ctx, f.source, &actual)
	actual.javaType = typ
	if !known || actual.externalName != "java.util.List" || actual.nominalScope != nil || actual.parameter != nil || actual.arrayRank != 0 || len(actual.arguments) != 1 || !actual.arguments[0].wildcard || actual.arguments[0].externalName != "java.lang.Integer" || actual.arguments[0].nominalScope != nil || actual.arguments[0].parameter != nil {
		t.Fatal("unmutated canonical upper-capture actual premise absent")
	}
	declaring, _ := genericMethodOwnerContext(f.method)
	declaring.localScope = f.method
	binder := f.method.TypeParameters[0]
	if binder.Declaration == nil || len(binder.Bounds) != 1 || f.method.Parameters[0].TypeParameterBindings[binder.Name] != binder.Declaration || !genericMethodClassHeaderOwned(f.method, declaring.currentFile.Source) {
		t.Fatal("unmutated owned method formal/header/binder premise absent")
	}
	bound := declaredJavaTypeOrigin(binder.Bounds[0], declaring)
	formals := methodFormalOrigins(f.method, f.owner, nil, f.ctx)
	if bound.externalName != "java.lang.Number" || bound.nominalScope != nil || bound.parameter != nil || bound.unresolved || len(formals) != 1 || formals[0].externalName != "java.util.Collection" || formals[0].nominalScope != nil || len(formals[0].arguments) != 1 || formals[0].arguments[0].parameter != binder.Declaration {
		t.Fatal("unmutated canonical callee formal/bound premise absent")
	}
	constraints, ok := methodInvocationConstraintBindings(f.method, f.owner, f.args, false, f.ctx, f.source)
	if !ok || len(constraints) != 1 || constraints[binder.Name] != "java.lang.Integer" {
		t.Fatal("unmutated capture constraint premise absent")
	}
	score, ok := scoreDeclaredMethodCandidate(f.method, f.owner, methodCandidateTypeParameterNames(f.owner, f.method), f.args, f.ctx, f.source, nil, nil)
	if !ok || score.phase != 0 || score.exactCount != 0 || (score.totalCost != 17 && score.totalCost != 1) {
		t.Fatalf("unmutated strict source candidate premise changed: applies=%v score=%+v", ok, score)
	}
	return score
}

func wildcardStrictScoreReplaceRegisteredFile(t *testing.T, previous, next *symbol.FileScope) {
	t.Helper()
	pkg := symbol.GlobalScope.Packages[previous.Package]
	if pkg == nil {
		t.Fatal("registered package absent")
	}
	count := 0
	for name, file := range pkg.Files {
		if file == previous {
			pkg.Files[name] = next
			count++
		}
	}
	if count != 1 {
		t.Fatalf("exact registered file premise=%d", count)
	}
}

func TestDeclaredMethodWildcardAuthorityFixtureWidening(t *testing.T) {
	for _, name := range []string{"original_upper_authority", "equal_content_caller_slice", "equal_content_callee_slice", "unicode_caller_input", "unicode_callee_bound_input"} {
		t.Run(name, func(t *testing.T) {
			header, locals := "<E extends Number>", "List<? extends Integer> values=null;"
			if name == "unicode_caller_input" {
				locals = `/*\uu0061*/ List<? extends Integer> values=null;`
			}
			if name == "unicode_callee_bound_input" {
				header = `/*\uu0061*/ <E extends Number>`
			}
			f := wildcardStrictScoreSetup(t, header, "Collection<E> values", locals, "values", nil)
			if name == "equal_content_caller_slice" {
				f.ctx.currentFile.Source = append([]byte(nil), f.ctx.currentFile.Source...)
				f.source = f.ctx.currentFile.Source
			}
			if name == "equal_content_callee_slice" {
				ctx, _ := genericMethodOwnerContext(f.method)
				ctx.currentFile.Source = append([]byte(nil), ctx.currentFile.Source...)
			}
			score := wildcardStrictScoreOwnedBeforeMutation(t, f)
			t.Logf("WILDCARD_STRICT_AUTHORITY_FACT case=%s callerInput=true calleeInput=true ownedCanonicalUpper=true bounds=true score=%+v", name, score)
			if score.totalCost != 1 {
				t.Fatalf("WILDCARD_STRICT_AUTHORITY: %s owned strict parent widening must cost1, got %+v", name, score)
			}
		})
	}
}

func TestDeclaredMethodWildcardStrictScorerRefusals(t *testing.T) {
	const upper = "List<? extends Integer> values=null;"
	for _, row := range []struct {
		name, header, formal, locals, arguments string
		extra                                   map[string]string
		mutate                                  func(*testing.T, *wildcardStrictScoreFixture)
		namespace                               string
		prepare                                 func(*testing.T, *wildcardStrictScoreFixture)
		boundsRefusal                           bool
	}{

		{name: "local_type_List", namespace: "local:List", locals: "class List<T>{} List<? extends Integer> values=null;"},
		{name: "local_type_Integer", namespace: "local:Integer", locals: "class Integer{} List<? extends Integer> values=null;"},
		{name: "local_type_java", namespace: "local:java", locals: "class java{} java.util.List<? extends java.lang.Integer> values=null;"},
		{name: "caller_inherited_List", namespace: "caller-parent", extra: map[string]string{"probe/Main.java": `package probe;import java.util.List;import java.lang.Integer;import java.lang.Number;class Parent{static class List<T>{}}public class Main extends Parent{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}, prepare: func(t *testing.T, f *wildcardStrictScoreFixture) { f.ctx.currentClass.Superclass = "" }},
		{name: "callee_inherited_Collection", namespace: "callee-parent", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;class Parent{static class Collection<T>{}}public class Selectors extends Parent{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}, prepare: func(t *testing.T, f *wildcardStrictScoreFixture) { f.owner.Superclass = "" }},
		{name: "callee_inherited_Number", namespace: "callee-parent", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;class Parent{static class Number{}}public class Selectors extends Parent{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}, prepare: func(t *testing.T, f *wildcardStrictScoreFixture) { f.owner.Superclass = "" }},
		{name: "static_type_List", namespace: "caller-import", extra: map[string]string{"probe/Main.java": `package probe;import java.util.List;import java.lang.Integer;import java.lang.Number;import static foreign.Owner.List;public class Main{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}},
		{name: "static_type_Number", namespace: "callee-import", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;import static foreign.Owner.Number;public class Selectors{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "static_demand_import", namespace: "caller-import", extra: map[string]string{"probe/Main.java": `package probe;import java.util.List;import java.lang.Integer;import java.lang.Number;import static foreign.Owner.*;public class Main{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}},
		{name: "ordinary_demand_import", namespace: "caller-import", extra: map[string]string{"probe/Main.java": `package probe;import java.util.List;import java.lang.Integer;import java.lang.Number;import foreign.*;public class Main{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}},
		{name: "member_type_Collection", namespace: "callee-member:Collection", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;public class Selectors{static class Collection<T>{}public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "member_type_Number", namespace: "callee-member:Number", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import java.lang.Number;public class Selectors{static class Number{}public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "duplicate_List_foreign_first", namespace: "caller-duplicate:List", extra: map[string]string{"probe/Main.java": `package probe;import foreign.List;import java.util.List;import java.lang.Integer;import java.lang.Number;public class Main{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}},
		{name: "duplicate_List_foreign_last", namespace: "caller-duplicate:List", extra: map[string]string{"probe/Main.java": `package probe;import java.util.List;import foreign.List;import java.lang.Integer;import java.lang.Number;public class Main{public static <Q extends Number> int run(){List<? extends Integer> values=null;return library.Selectors.select(values);}}`}},
		{name: "duplicate_Collection_foreign_first", namespace: "callee-duplicate:Collection", extra: map[string]string{"library/Selectors.java": `package library;import foreign.Collection;import java.util.Collection;import java.lang.Number;public class Selectors{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "duplicate_Collection_foreign_last", namespace: "callee-duplicate:Collection", extra: map[string]string{"library/Selectors.java": `package library;import java.util.Collection;import foreign.Collection;import java.lang.Number;public class Selectors{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "duplicate_Number_foreign_first", namespace: "callee-duplicate:Number", extra: map[string]string{"library/Selectors.java": `package library;import foreign.Number;import java.lang.Number;import java.util.Collection;public class Selectors{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "duplicate_Number_foreign_last", namespace: "callee-duplicate:Number", extra: map[string]string{"library/Selectors.java": `package library;import java.lang.Number;import foreign.Number;import java.util.Collection;public class Selectors{public static <E extends Number> int select(Collection<E> values){return 1;}public static int select(Object value){return 2;}}`}},
		{name: "caller_source_stale_before_query", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			input := f.ctx.currentFile.SourceInputSnapshot()
			f.ctx.currentFile.Source = append(append([]byte(nil), f.ctx.currentFile.Source...), ' ')
			f.source = f.ctx.currentFile.Source
			if input.Validate() == nil {
				t.Fatal("caller attachment did not observe stale bytes")
			}
		}},
		{name: "callee_source_stale_before_query", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			input := ctx.currentFile.SourceInputSnapshot()
			ctx.currentFile.Source = append(append([]byte(nil), ctx.currentFile.Source...), ' ')
			if input.Validate() == nil {
				t.Fatal("callee attachment did not observe stale bytes")
			}
		}},
		{name: "caller_filename_stale", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			input := f.ctx.currentFile.SourceInputSnapshot()
			f.ctx.currentFile.SourceFilename += ".other"
			if input.Validate() == nil {
				t.Fatal("caller registered-name mutation absent")
			}
		}},
		{name: "callee_filename_stale", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			input := ctx.currentFile.SourceInputSnapshot()
			ctx.currentFile.SourceFilename += ".other"
			if input.Validate() == nil {
				t.Fatal("callee registered-name mutation absent")
			}
		}},
		{name: "copied_caller_file", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			copy := *f.ctx.currentFile
			if copy.ParsedRoot() != f.ctx.currentFile.ParsedRoot() || copy.SourceInputSnapshot() != nil {
				t.Fatal("copied owner input premise absent")
			}
			f.ctx.currentFile = &copy
		}},
		{name: "copied_callee_file", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			copy := *ctx.currentFile
			if copy.ParsedRoot() != ctx.currentFile.ParsedRoot() || copy.SourceInputSnapshot() != nil {
				t.Fatal("copied callee input premise absent")
			}
			wildcardStrictScoreReplaceRegisteredFile(t, ctx.currentFile, &copy)
		}},
		{name: "caller_missing_attachment", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			old := f.ctx.currentFile
			copy := symbol.ParseSymbols(old.ParsedRoot(), old.Source)
			copy.SourceFilename = old.SourceFilename
			if copy.SourceInputSnapshot() != nil || old.SourceInputSnapshot().Validate() != nil {
				t.Fatal("missing attachment/valid old input premise absent")
			}
			wildcardStrictScoreReplaceRegisteredFile(t, old, copy)
			f.ctx.currentFile = copy
			f.ctx.currentClass = copy.FindClassScope("Main")
			f.caller = f.ctx.currentClass.FindMethodByName("run", nil)
			f.ctx.localScope = f.caller
		}},
		{name: "callee_missing_attachment", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			old := ctx.currentFile
			copy := symbol.ParseSymbols(old.ParsedRoot(), old.Source)
			copy.SourceFilename = old.SourceFilename
			if copy.SourceInputSnapshot() != nil || old.SourceInputSnapshot().Validate() != nil {
				t.Fatal("missing callee attachment/valid old input premise absent")
			}
			wildcardStrictScoreReplaceRegisteredFile(t, old, copy)
			f.owner = copy.FindClassScope("Selectors")
			for _, m := range f.owner.Methods {
				if len(m.TypeParameters) == 1 {
					f.method = m
				}
			}
		}},
		{name: "callee_graph_owner_removed", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			pkg := symbol.GlobalScope.Packages[ctx.currentFile.Package]
			found := false
			for key, file := range pkg.Files {
				if file == ctx.currentFile {
					delete(pkg.Files, key)
					found = true
				}
			}
			if !found || ctx.currentFile.SourceInputSnapshot().Validate() != nil {
				t.Fatal("graph absence must be separate from still-valid input")
			}
		}},
		{name: "caller_unicode_original_stale", locals: `/*\uu0061*/ List<? extends Integer> values=null;`, mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			file := f.ctx.currentFile
			input := file.SourceInputSnapshot()
			if len(file.OriginalSource) == 0 || len(file.SourceOffsets) == 0 {
				t.Fatal("actual Unicode input domain absent")
			}
			file.OriginalSource = append(append([]byte(nil), file.OriginalSource...), ' ')
			if input.Validate() == nil {
				t.Fatal("Unicode caller original-byte mutation absent")
			}
		}},
		{name: "caller_unicode_offsets_stale", locals: `/*\uu0061*/ List<? extends Integer> values=null;`, mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			file := f.ctx.currentFile
			input := file.SourceInputSnapshot()
			if len(file.OriginalSource) == 0 || len(file.SourceOffsets) == 0 {
				t.Fatal("actual Unicode map absent")
			}
			file.SourceOffsets = append(append([]uint32(nil), file.SourceOffsets...), 0)
			if input.Validate() == nil {
				t.Fatal("Unicode caller map mutation absent")
			}
		}},
		{name: "callee_unicode_original_stale", header: `/*\uu0061*/ <E extends Number>`, mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			file := ctx.currentFile
			input := file.SourceInputSnapshot()
			if len(file.OriginalSource) == 0 || len(file.SourceOffsets) == 0 {
				t.Fatal("actual Unicode callee domain absent")
			}
			file.OriginalSource = append(append([]byte(nil), file.OriginalSource...), ' ')
			if input.Validate() == nil {
				t.Fatal("Unicode callee original mutation absent")
			}
		}},
		{name: "callee_unicode_offsets_stale", header: `/*\uu0061*/ <E extends Number>`, mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			file := ctx.currentFile
			input := file.SourceInputSnapshot()
			if len(file.OriginalSource) == 0 || len(file.SourceOffsets) == 0 {
				t.Fatal("actual Unicode callee map absent")
			}
			file.SourceOffsets = append(append([]uint32(nil), file.SourceOffsets...), 0)
			if input.Validate() == nil {
				t.Fatal("Unicode callee map mutation absent")
			}
		}},
		{name: "non_identifier_occurrence", arguments: "(values)"},
		{name: "intersection_bound", header: "<E extends Number & Comparable<E>>"},
		{name: "duplicate_local_name", locals: "List<? extends Integer> values=null;{List<? extends Integer> values=null;}"},
		{name: "foreign_argument_tree", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			foreign := wildcardStrictScoreForeign(t, f.source, "Main")
			var call *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n.Type() == "method_invocation" {
					name := n.ChildByFieldName("name")
					if name != nil && name.Content(f.source) == "select" {
						call = n
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(n) {
					walk(child)
				}
			}
			walk(foreign.FindMethodByName("run", nil).DeclarationNode)
			if call == nil {
				t.Fatal("foreign call absent")
			}
			f.args = nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
			if threadSourceNodeWithin(f.args[0], f.ctx.currentFile.ParsedRoot()) {
				t.Fatal("foreign occurrence not distinct")
			}
		}},
		{name: "foreign_source_bytes", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			f.source = append(append([]byte(nil), f.source...), ' ')
			if len(f.source) == len(f.ctx.currentFile.Source) {
				t.Fatal("source mismatch absent")
			}
		}},
		{name: "caller_package_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) { f.ctx.currentFile.Package = "foreign" }},
		{name: "callee_package_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			ctx.currentFile.Package = "foreign"
		}},
		{name: "same_canonical_owner", locals: "Collection<? extends Integer> values=null;"},
		{name: "actual_array_rank", locals: "List<? extends Integer>[] values=null;"},
		{name: "formal_array_rank", formal: "Collection<E>[] values", locals: "List<? extends Integer>[] values=null;"},
		{name: "nested_actual_argument", locals: "List<? extends List<Integer>> values=null;", boundsRefusal: true},
		{name: "actual_post_declarator_rank", locals: "List<? extends Integer> values[]=null;"},
		{name: "formal_post_declarator_rank", formal: "Collection<E> values[]"},
		{name: "lower_nested_concrete", header: "<E>", formal: "Collection<? super E> values", locals: "List<List<String>> values=null;"},
		{name: "nested_formal_argument", formal: "Collection<List<E>> values", locals: "List<? extends List<Integer>> values=null;"},
		{name: "lower_capture_to_invariant", header: "<E>", locals: "List<? super Integer> values=null;"},
		{name: "upper_capture_to_lower", header: "<E>", formal: "Collection<? super E> values"},
		{name: "incompatible_declared_bound", locals: "List<String> values=null;", boundsRefusal: true},
		{name: "concrete_invariant_formal", formal: "Collection<Integer> values"},
		{name: "wrong_invariant_argument", formal: "Collection<Integer> values", locals: "List<String> values=null;"},
		{name: "uncertified_parent", locals: "java.util.Comparator<? extends Integer> values=null;"},
		{name: "source_actual_owner", locals: "library.LocalList<? extends Integer> values=null;", extra: map[string]string{"library/LocalList.java": `package library;public class LocalList<T> extends java.util.ArrayList<T>{}`}},
		{name: "source_formal_owner", formal: "library.LocalCollection<E> values", extra: map[string]string{"library/LocalCollection.java": `package library;public interface LocalCollection<T>{}`}},
		{name: "source_argument_homonym", locals: "List<? extends library.Integer> values=null;", extra: map[string]string{"library/Integer.java": `package library;public class Integer extends java.lang.Number{}`}},
		{name: "source_bound_homonym", header: "<E extends library.Number>", extra: map[string]string{"library/Number.java": `package library;public class Number{}`}, boundsRefusal: true},
		{name: "actual_import_table_extra", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			f.ctx.currentFile.Imports["Unused"] = "foreign"
			if f.ctx.currentFile.Imports["Unused"] != "foreign" {
				t.Fatal("source-absent actual import mutation missing")
			}
		}},
		{name: "callee_import_table_extra", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			ctx.currentFile.Imports["Unused"] = "foreign"
			if ctx.currentFile.Imports["Unused"] != "foreign" {
				t.Fatal("source-absent callee import mutation missing")
			}
		}},
		{name: "foreign_actual_import", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			f.ctx.currentFile.Imports["List"] = "foreign"
			if f.ctx.currentFile.Imports["List"] != "foreign" {
				t.Fatal("import mutation absent")
			}
		}},
		{name: "foreign_formal_import", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			ctx.currentFile.Imports["Collection"] = "foreign"
			if ctx.currentFile.Imports["Collection"] != "foreign" {
				t.Fatal("import mutation absent")
			}
		}},
		{name: "foreign_caller_tree", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			foreign := wildcardStrictScoreForeign(t, f.source, "Main")
			f.ctx.currentClass.Class.DeclarationNode = foreign.Class.DeclarationNode
			if threadSourceNodeWithin(f.ctx.currentClass.Class.DeclarationNode, f.ctx.currentFile.ParsedRoot()) {
				t.Fatal("foreign caller tree not distinct")
			}
		}},
		{name: "foreign_method_tree", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			foreign := wildcardStrictScoreForeign(t, ctx.currentFile.Source, "Selectors")
			for _, m := range foreign.Methods {
				if len(m.TypeParameters) > 0 {
					f.method.DeclarationNode = m.DeclarationNode
				}
			}
			if threadSourceNodeWithin(f.method.DeclarationNode, ctx.currentFile.ParsedRoot()) {
				t.Fatal("foreign method tree not distinct")
			}
		}},
		{name: "foreign_formal_binder", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			p := symbol.NewTypeParam("E", nil)
			f.method.Parameters[0].TypeParameterBindings["E"] = p.Declaration
			if f.method.Parameters[0].TypeParameterBindings["E"] == f.method.TypeParameters[0].Declaration {
				t.Fatal("foreign formal binder not distinct")
			}
		}},
		{name: "foreign_actual_binder", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			p := f.caller.TypeParameters[0].Declaration
			f.caller.FindVariable("values").TypeParameterBindings["Integer"] = p
			if p == f.method.TypeParameters[0].Declaration {
				t.Fatal("foreign actual binder not distinct")
			}
		}},
		{name: "caller_type_parameter", locals: "List<? extends Q> values=null;"},
		{name: "replaced_method_binder", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			old := f.method.TypeParameters[0].Declaration
			f.method.TypeParameters[0].Declaration = symbol.NewTypeParam("E", nil).Declaration
			if f.method.TypeParameters[0].Declaration == old || f.method.Parameters[0].TypeParameterBindings["E"] != old {
				t.Fatal("method binder replacement premise absent")
			}
		}},
		{name: "formal_TYPE_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			f.method.Parameters[0].OriginalType = "Collection<? super E>"
			if f.method.DeclarationNode.ChildByFieldName("parameters").NamedChild(0).ChildByFieldName("type").Content(ctx.currentFile.Source) == f.method.Parameters[0].OriginalType {
				t.Fatal("TYPE mismatch absent")
			}
		}},
		{name: "formal_name_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) { f.method.Parameters[0].OriginalName = "other" }},
		{name: "callee_bound_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			ctx, _ := genericMethodOwnerContext(f.method)
			f.method.TypeParameters[0].Bounds = nil
			if genericMethodClassHeaderOwned(f.method, ctx.currentFile.Source) {
				t.Fatal("header mismatch absent")
			}
		}},
		{name: "actual_TYPE_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			f.caller.FindVariable("values").OriginalType = "List<? extends Number>"
		}},
		{name: "actual_name_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) {
			f.caller.FindVariable("values").OriginalName = "other"
			if f.caller.FindVariable("values") != nil {
				t.Fatal("name mismatch absent")
			}
		}},
		{name: "explicit_invocation_type_argument", arguments: "<Integer>:values"},
		{name: "variable_arity", formal: "Collection<E>... values"},
		{name: "two_method_binders", header: "<E extends Number,F>"},
		{name: "two_formals", formal: "Collection<E> values,Object extra", arguments: "values,null"},
		{name: "zero_arguments", arguments: "<empty>"},
		{name: "zero_formal_metadata", mutate: func(t *testing.T, f *wildcardStrictScoreFixture) { f.method.Parameters = nil }},
		{name: "independent_same_upper_captures", formal: "Collection<E> values,Collection<E> other", locals: upper + "List<? extends Integer> other=null;", arguments: "values,other", boundsRefusal: true},
		{name: "concrete_insertion_into_capture", formal: "Collection<E> values,E extra", arguments: "values,1", boundsRefusal: true},
		{name: "conflicting_invariant_arguments", formal: "Collection<E> values,Collection<E> other", locals: "List<Integer> values=null;List<Double> other=null;", arguments: "values,other", boundsRefusal: true},
	} {
		t.Run(row.name, func(t *testing.T) {
			header, formal, locals, arguments := row.header, row.formal, row.locals, row.arguments
			if header == "" {
				header = "<E extends Number>"
			}
			if formal == "" {
				formal = "Collection<E> values"
			}
			if locals == "" {
				locals = upper
			}
			if arguments == "" {
				arguments = "values"
			}
			if arguments == "<empty>" {
				arguments = ""
			}
			f := wildcardStrictScoreSetup(t, header, formal, locals, arguments, row.extra)
			if row.mutate != nil {
				before := wildcardStrictScoreOwnedBeforeMutation(t, f)
				t.Logf("WILDCARD_STRICT_PREMUTATION case=%s score=%+v", row.name, before)
				row.mutate(t, f)
			}
			if row.namespace != "" {
				wildcardStrictScoreFreshInputs(t, f)
				wildcardStrictScoreNamespacePremise(t, f, row.namespace)
				if row.prepare != nil {
					row.prepare(t, f)
				}
				wildcardStrictScoreFreshInputs(t, f)
				t.Logf("WILDCARD_STRICT_NAMESPACE case=%s inputCurrent=true originalAST=true", row.name)
			}
			if row.boundsRefusal {
				if _, ok := methodInvocationConstraintBindings(f.method, f.owner, f.args, false, f.ctx, f.source); ok {
					t.Fatal("intended invariant/capture/bound constraint refusal absent")
				}
			}
			score, applies := scoreDeclaredMethodCandidate(f.method, f.owner, methodCandidateTypeParameterNames(f.owner, f.method), f.args, f.ctx, f.source, nil, nil)
			t.Logf("WILDCARD_STRICT_REFUSAL case=%s actualArgs=%d formals=%d binders=%d applies=%v score=%+v", row.name, len(f.args), len(f.method.Parameters), len(f.method.TypeParameters), applies, score)
			if applies && score.phase == 0 && score.totalCost < 2 {
				t.Fatalf("unsupported original scorer shape received strict wildcard widening credit: %+v", score)
			}
		})
	}
}

func TestDeclaredMethodWildcardLooseAndObjectScores(t *testing.T) {
	for _, row := range []struct {
		name, formal, locals string
		want                 int
	}{
		{"loose_unboxing_stays_existing", "int value", "Integer values=null;", 8},
		{"canonical_Object_fallback", "Object value", "Integer values=null;", 2},
		{"unbounded_actual_wildcard", "Collection<? super E> values", "List<?> values=null;", 1},
		{"lower_actual_wildcard", "Collection<? super E> values", "List<? super Integer> values=null;", 1},
	} {
		t.Run(row.name, func(t *testing.T) {
			f := wildcardStrictScoreSetup(t, "<E>", row.formal, row.locals, "values", nil)
			if row.name == "loose_unboxing_stays_existing" {
				_, _, strict := javaInvocationConversionCost(f.args[0], "int", []string{"E"}, f.ctx, f.source)
				_, _, loose := javaLooseInvocationConversionCost(f.args[0], "int", []string{"E"}, f.ctx, f.source)
				if strict || !loose {
					t.Fatal("actual strict-refusal/loose-applicability premise absent")
				}
			}
			score, ok := scoreDeclaredMethodCandidate(f.method, f.owner, methodCandidateTypeParameterNames(f.owner, f.method), f.args, f.ctx, f.source, nil, nil)
			t.Logf("WILDCARD_STRICT_EXISTING_SCORE case=%s applies=%v score=%+v", row.name, ok, score)
			if !ok || score.totalCost != row.want || score.phase != 0 || score.exactCount != 0 {
				t.Fatalf("existing score changed: applies=%v score=%+v wantCost=%d", ok, score, row.want)
			}
		})
	}
}

func TestDeclaredMethodWildcardSourceAmbiguityScores(t *testing.T) {
	root := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Types.java": `package p;interface A<T>{}interface B<T>{}class Both<T> implements A<T>,B<T>{}class Owner{static <E>int pick(A<E> value){return 1;}static <E>int pick(B<E> value){return 2;}}class Use{static int run(){Both<? extends Integer> value=null;return Owner.pick(value);}}`,
	})
	owner, callerOwner := findQualifiedSourceClass("p.Owner"), findQualifiedSourceClass("p.Use")
	if owner == nil || callerOwner == nil {
		t.Fatal("source ambiguity owners absent")
	}
	ctx := classScopeCtx(callerOwner, root)
	ctx.localScope = callerOwner.FindMethodByName("run", nil)
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name")
			if name != nil && name.Content(ctx.currentFile.Source) == "pick" {
				call = n
			}
		}
		for _, c := range nodeutil.NamedChildrenOf(n) {
			walk(c)
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if call == nil || len(owner.Methods) != 2 {
		t.Fatal("two exact source candidates/invocation absent")
	}
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	scores := make([]methodCandidateScore, 2)
	for i, method := range owner.Methods {
		score, ok := scoreDeclaredMethodCandidate(method, owner, methodCandidateTypeParameterNames(owner, method), args, ctx, ctx.currentFile.Source, nil, nil)
		if !ok || score.totalCost != 1 || score.phase != 0 {
			t.Fatalf("source candidate premise changed: applies=%v score=%+v", ok, score)
		}
		scores[i] = score
	}
	if methodCandidateScoreBetter(scores[0], scores[1]) || methodCandidateScoreBetter(scores[1], scores[0]) {
		t.Fatal("unrelated source candidates gained a wildcard score preference")
	}
}

// These retained grammar witnesses deliberately remain valid when mutable
// hierarchy/import lookup state cannot certify the TYPE namespace.
func wildcardStrictScoreNamespacePremise(t *testing.T, f *wildcardStrictScoreFixture, kind string) {
	t.Helper()
	declaring, _ := genericMethodOwnerContext(f.method)
	if strings.HasPrefix(kind, "local:") {
		name := strings.TrimPrefix(kind, "local:")
		if !sourceGenericViewLocalTypeShadowed(f.args[0], name, f.source) {
			t.Fatal("original local TYPE shadow absent")
		}
		return
	}
	ctx, occurrence := f.ctx, f.caller.DeclarationNode
	if strings.HasPrefix(kind, "callee-") {
		ctx, occurrence = declaring, f.method.DeclarationNode
	}
	if strings.Contains(kind, "-duplicate:") {
		name := strings.SplitN(kind, ":", 2)[1]
		matches := 0
		for _, node := range nodeutil.SemanticNamedChildrenOf(ctx.currentFile.ParsedRoot()) {
			if node.Type() != "import_declaration" {
				continue
			}
			names := nodeutil.SemanticNamedChildrenOf(node)
			if len(names) != 1 {
				continue
			}
			n := names[0].ChildByFieldName("name")
			if n != nil && n.Content(ctx.currentFile.Source) == name {
				matches++
			}
		}
		if matches != 2 {
			t.Fatal("two original conflicting TYPE imports absent")
		}
		return
	}
	if strings.HasSuffix(kind, "-parent") {
		found := false
		for ancestor := occurrence.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
			if sourceGenericViewTypeDeclaration(ancestor) && ancestor.ChildByFieldName("superclass") != nil {
				found = true
			}
		}
		if !found {
			t.Fatal("original written parent absent")
		}
		return
	}
	if strings.Contains(kind, "-member:") {
		name := strings.SplitN(kind, ":", 2)[1]
		if !classServiceASTTypePrefix(occurrence, name, ctx.currentFile.Source, ctx) {
			t.Fatal("original member TYPE absent")
		}
		return
	}
	found := false
	for _, node := range nodeutil.SemanticNamedChildrenOf(ctx.currentFile.ParsedRoot()) {
		if node.Type() != "import_declaration" {
			continue
		}
		for index := 0; index < int(node.ChildCount()); index++ {
			child := node.Child(index)
			found = found || child.Content(ctx.currentFile.Source) == "*" || child.Type() == "static"
		}
	}
	if !found {
		t.Fatal("original unresolved static/demand import absent")
	}
}
