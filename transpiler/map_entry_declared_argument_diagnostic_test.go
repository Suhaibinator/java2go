package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestMapEntryActualSourceArgumentDeclarationSelection(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	var files []parsing.SourceFile
	for _, row := range mapEntryActualArgumentFiles {
		f := parsing.SourceFile{Name: row.name, Source: []byte(row.java)}
		if err := f.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(f.ParseSymbols())
		files = append(files, f)
	}
	for _, f := range files {
		ResolveFile(f)
	}
	scope := findQualifiedSourceClass("probe.app.Main")
	file := findFileScopeForClassScope(scope)
	if scope == nil || file == nil {
		t.Fatal("actual caller declaration missing")
	}
	ctx := Ctx{currentClass: scope, currentFile: file, className: scope.Class.Name, localScope: scope.FindMethodByName("main", nil), executionContextName: "__actualInvocationExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	if ctx.localScope == nil {
		t.Fatal("actual main definition missing")
	}
	source := file.Source
	ParseStmt(ctx.localScope.DeclarationNode.ChildByFieldName("body"), source, ctx)
	var calls []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name")
			object := n.ChildByFieldName("object")
			if name != nil && name.Content(source) == "add" && object != nil && (object.Content(source) == "strings" || object.Content(source) == "objects") {
				calls = append(calls, n)
			}
		}
		for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
			walk(c)
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(calls) != 2 {
		t.Fatalf("actual add calls=%d", len(calls))
	}
	var describe func(inferredJavaTypeOrigin) any
	describe = func(o inferredJavaTypeOrigin) any {
		name := ""
		if o.nominalScope != nil {
			name = qualifiedSourceClassName(o.nominalScope)
		}
		args := []any{}
		for _, a := range o.arguments {
			args = append(args, describe(a))
		}
		return map[string]any{"java": o.javaType, "external": o.externalName, "nominal": name, "unresolved": o.unresolved, "parameter": o.parameter != nil, "arguments": args}
	}
	for _, call := range calls {
		object := call.ChildByFieldName("object")
		args := call.ChildByFieldName("arguments")
		target := resolveInvocationTarget(object, ctx, source)
		if target == nil || target.classScope == nil {
			t.Fatal("actual source receiver missing")
		}
		def := target.classScope.FindMethodByName("add", nil)
		if def == nil || len(def.Parameters) != 1 {
			t.Fatal("actual source add declaration missing")
		}
		actualNode := invocationArgumentNode(call, 0)
		var actual inferredJavaTypeOrigin
		typ, known := inferExprJavaType(actualNode, ctx, source, &actual)
		actual.javaType = typ
		formal := methodFormalOrigins(def, target.classScope, nil, ctx)[0]
		cost, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
		selected, _ := findBestMethodForInvocationTarget(target, "add", args, true, true, ctx, source)
		selectedName := ""
		if selected != nil && selected.def != nil {
			selectedName = selected.def.Name
		}
		t.Logf("call=%s receiver=%s sourceName=%s emittedName=%s formalOriginal=%s actualKnown=%t actual=%#v formal=%#v conversion=(%d,%t,%t,%t) selected=%s", call.Content(source), qualifiedSourceClassName(target.classScope), def.OriginalName, def.Name, def.Parameters[0].OriginalType, known, describe(actual), describe(formal), cost, exact, applies, handled, selectedName)
		declaring := classScopeCtx(target.classScope, ctx)
		t.Logf("declaringMapImport=%q nestedOwner=%q fullyQualifiedOwner=%q", declaring.currentFile.Imports["Map"], declaredExternalNominalOwner("Map.Entry", nil, declaring), declaredExternalNominalOwner("java.util.Map.Entry", nil, declaring))
		if selected == nil || selected.def != def || selected.owner != target.classScope {
			t.Errorf("actual public Java add declaration was not selected")
		}
	}
}

var mapEntryActualArgumentFiles = []struct{ name, java string }{
	{"src/main/java/probe/app/Main.java", "package probe.app;\nimport java.util.Map;\nimport probe.impl.Cell;\nimport probe.entry.Slot;\nimport probe.entry.Trace;\npublic final class Main {\n    public static void main(String[] args) {\n        probe.impl.Cells strings = new probe.impl.Cells();\n        Cell cell = new Cell(\"key\", \"old\");\n        System.out.println(\"add:\" + strings.add(cell) + \":\" + strings.size());\n        Map.Entry<String,String> alias = strings.iterator().next();\n        System.out.println(\"alias:\" + (alias == cell) + \":\" + alias.setValue(\"next\") + \":\" + cell.getValue());\n        probe.store.Cells objects = new probe.store.Cells();\n        Slot slot = new Slot(\"wide\", Integer.valueOf(7));\n        System.out.println(\"wide-add:\" + objects.add(slot) + \":\" + objects.size());\n        Map.Entry<String,Object> wide = objects.iterator().next();\n        Object old = wide.setValue(\"done\");\n        System.out.println(\"wide-alias:\" + (wide == slot) + \":\" + old + \":\" + slot.getValue());\n        System.out.println(\"trace:\" + Trace.take());\n    }\n}\n"},
	{"src/main/java/probe/entry/Slot.java", "package probe.entry;\nimport java.util.Map;\npublic final class Slot implements Map.Entry<String,Object> {\n    private final String key;\n    private Object value;\n    public Slot(String key, Object value) { this.key = key; this.value = value; }\n    public String getKey() { Trace.add(\"key\"); return key; }\n    public Object getValue() { Trace.add(\"value\"); return value; }\n    public Object setValue(Object next) { Trace.add(\"set\"); Object old = value; value = next; return old; }\n}\n"},
	{"src/main/java/probe/entry/Trace.java", "package probe.entry;\npublic final class Trace {\n    private static String events = \"\";\n    public static void add(String event) { events += event + \",\"; }\n    public static String take() { String found = events; events = \"\"; return found; }\n}\n"},
	{"src/main/java/probe/impl/Cell.java", "package probe.impl;\nimport java.util.Map;\npublic final class Cell implements Map.Entry<String,String> {\n    private final String key;\n    private String value;\n    public Cell(String key, String value) { this.key = key; this.value = value; }\n    public String getKey() { return key; }\n    public String getValue() { return value; }\n    public String setValue(String next) { String old = value; value = next; return old; }\n}\n"},
	{"src/main/java/probe/impl/Cells.java", "package probe.impl;\nimport java.util.AbstractSet;\nimport java.util.ArrayList;\nimport java.util.Iterator;\nimport java.util.Map;\npublic final class Cells extends AbstractSet<Map.Entry<String,String>> {\n    private final ArrayList<Map.Entry<String,String>> data = new ArrayList<>();\n    public Iterator<Map.Entry<String,String>> iterator() { return data.iterator(); }\n    public int size() { return data.size(); }\n    public boolean add(Map.Entry<String,String> cell) { return data.add(cell); }\n}\n"},
	{"src/main/java/probe/store/Cells.java", "package probe.store;\nimport java.util.AbstractSet;\nimport java.util.ArrayList;\nimport java.util.Iterator;\nimport java.util.Map;\nimport probe.entry.Trace;\npublic final class Cells extends AbstractSet<Map.Entry<String,Object>> {\n    private final ArrayList<Map.Entry<String,Object>> data = new ArrayList<>();\n    public boolean add(Map.Entry<String,Object> entry) { return data.add(entry); }\n    public int size() { return data.size(); }\n    public Iterator<Map.Entry<String,Object>> iterator() {\n        final Iterator<Map.Entry<String,Object>> cursor = data.iterator();\n        return new Iterator<Map.Entry<String,Object>>() {\n            public boolean hasNext() { Trace.add(\"has\"); return cursor.hasNext(); }\n            public Map.Entry<String,Object> next() { Trace.add(\"next\"); return cursor.next(); }\n            public void remove() { Trace.add(\"remove\"); cursor.remove(); }\n        };\n    }\n}\n"},
}
