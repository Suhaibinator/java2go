package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func enumSwitch184Original5() map[string]string {
	return map[string]string{
		"campaign/cache/api/Access.java":        "package campaign.cache.api;\n\npublic interface Access {\n    String label();\n    Object payload();\n    void touch(int delta);\n    int touches();\n    long credit();\n}\n",
		"campaign/cache/api/Lane.java":          "package campaign.cache.api;\n\npublic final class Lane<T> {\n    public enum Band { SMALL, NORMAL }\n    private final boolean direct;\n    private final int subpages;\n    private final T marker;\n    public Lane(boolean direct, int subpages, T marker) {\n        this.direct = direct;\n        this.subpages = subpages;\n        this.marker = marker;\n    }\n    public boolean isDirect() { return direct; }\n    public int subpages() { return subpages; }\n    public T marker() { return marker; }\n}\n",
		"campaign/cache/app/Main.java":          "package campaign.cache.app;\n\nimport campaign.cache.api.Access;\nimport campaign.cache.api.Lane;\nimport campaign.cache.api.Lane.Band;\nimport campaign.cache.store.Registry;\nimport java.nio.ByteBuffer;\nimport java.nio.charset.StandardCharsets;\nimport java.nio.file.Files;\nimport java.nio.file.Path;\n\npublic final class Main {\n    private static void emit(StringBuilder journal, String line) {\n        journal.append(line).append('\\n');\n        System.out.println(line);\n    }\n    private static String describe(Access access) {\n        if (access == null) { return \"null\"; }\n        Object payload = access.payload();\n        String first;\n        if (payload == null) { first = \"null\"; }\n        else if (payload instanceof byte[]) { first = \"bytes:\" + ((byte[]) payload)[0]; }\n        else { first = \"buffer:\" + ((ByteBuffer) payload).get(0); }\n        return access.label() + \":\" + access.touches() + \":\" + access.credit() + \":\" + first;\n    }\n    private static String attempt(Registry registry, Lane<?> lane, int index, Band band) {\n        try {\n            return describe(registry.find(lane, index, band));\n        } catch (RuntimeException failure) {\n            return failure.getClass().getName() + \":message=\" + failure.getMessage();\n        }\n    }\n    public static void main(String[] args) throws Exception {\n        int seed = Integer.parseInt(args[0]);\n        StringBuilder journal = new StringBuilder();\n        Lane<byte[]> heap = new Lane<>(false, 2, new byte[] {(byte) seed});\n        Lane<ByteBuffer> direct = new Lane<>(true, 2, ByteBuffer.wrap(new byte[] {(byte) seed}));\n        Registry registry = new Registry(seed, 2, true, true);\n        Access hs = registry.borrow(heap, 0, Band.SMALL, seed);\n        Access ds = registry.borrow(direct, 0, Band.SMALL, seed * 2);\n        Access hn = registry.borrow(heap, 2, Band.NORMAL, seed * 3);\n        Access dn = registry.borrow(direct, 2, Band.NORMAL, seed * 4);\n        emit(journal, \"seed=\" + seed + \":\" + describe(hs) + \":\" + describe(ds));\n        emit(journal, describe(hn) + \":\" + describe(dn));\n        emit(journal, \"identity=\" + (hs == registry.find(heap, 0, Band.SMALL))\n                + \":explicit=\" + (hs == registry.explicitHeap(0))\n                + \":buckets=\" + (hs != ds) + \":\" + (hs != hn) + \":\" + (ds != dn));\n        byte[] oldHeap = (byte[]) hs.payload();\n        ByteBuffer oldDirect = (ByteBuffer) ds.payload();\n        oldHeap[0]++;\n        oldDirect.put(0, (byte) (oldDirect.get(0) + 5));\n        hs.touch(-seed);\n        ds.touch(-seed);\n        emit(journal, \"mutated=\" + describe(registry.find(heap, 0, Band.SMALL))\n                + \":\" + describe(registry.find(direct, 0, Band.SMALL)));\n        byte[] replacement = new byte[] {(byte) (seed + 9), 9};\n        emit(journal, \"replaceHeapOld=\" + (registry.replaceHeap(0, replacement) == oldHeap)\n                + \":newAlias=\" + (hs.payload() == replacement) + \":sameSlot=\" + (hs == registry.explicitHeap(0)));\n        emit(journal, \"replaceDirectOld=\" + (registry.replaceDirect(0, null) == oldDirect)\n                + \":\" + describe(ds));\n        ByteBuffer replacementDirect = ByteBuffer.wrap(new byte[] {(byte) (seed + 11), 11});\n        emit(journal, \"replaceDirectNull=\" + (registry.replaceDirect(0, replacementDirect) == null)\n                + \":newAlias=\" + (ds.payload() == replacementDirect));\n        emit(journal, \"sparse=\" + attempt(registry, heap, 1, Band.SMALL)\n                + \":\" + attempt(registry, direct, 3, Band.NORMAL));\n        emit(journal, \"high=\" + attempt(registry, heap, 2, Band.SMALL)\n                + \":\" + attempt(registry, direct, 4, Band.NORMAL));\n        emit(journal, \"negative=\" + attempt(registry, direct, -1, Band.SMALL));\n        emit(journal, \"normal-low=\" + attempt(registry, heap, 1, Band.NORMAL));\n        emit(journal, \"null-lane=\" + attempt(registry, null, 0, Band.SMALL));\n        Registry disabled = new Registry(seed, 2, false, false);\n        emit(journal, \"null-array=\" + attempt(disabled, direct, -1, Band.SMALL)\n                + \":\" + attempt(disabled, heap, 99, Band.NORMAL));\n        Registry empty = new Registry(seed, 0, true, true);\n        emit(journal, \"empty-high=\" + attempt(empty, heap, 0, Band.SMALL));\n        emit(journal, \"empty-negative=\" + attempt(empty, direct, -1, Band.SMALL));\n        emit(journal, \"final=\" + describe(hs) + \":\" + describe(ds)\n                + \":normal=\" + describe(hn) + \":\" + describe(dn));\n        emit(journal, registry.snapshot() + \":disabled=\" + disabled.snapshot() + \":empty=\" + empty.snapshot());\n        Files.createDirectories(Path.of(\"out\"));\n        Files.writeString(Path.of(\"out/cache-journal.txt\"), journal.toString(), StandardCharsets.UTF_8);\n        System.out.println(\"done=complete\");\n    }\n}\n",
		"campaign/cache/store/Registry.java":    "package campaign.cache.store;\n\nimport campaign.cache.api.Access;\nimport campaign.cache.api.Lane;\nimport campaign.cache.api.Lane.Band;\nimport java.nio.ByteBuffer;\nimport static campaign.cache.validation.Checks.requireNonNegative;\n\npublic final class Registry {\n    private final Slot<byte[]>[] smallHeap;\n    private final Slot<ByteBuffer>[] smallDirect;\n    private final Slot<byte[]>[] normalHeap;\n    private final Slot<ByteBuffer>[] normalDirect;\n    private int routes;\n\n    @SuppressWarnings({\"unchecked\", \"rawtypes\"})\n    public Registry(int seed, int requestedSize, boolean heap, boolean direct) {\n        int size = requireNonNegative(requestedSize);\n        smallHeap = heap ? new Slot[size] : null;\n        normalHeap = heap ? new Slot[size] : null;\n        smallDirect = direct ? new Slot[size] : null;\n        normalDirect = direct ? new Slot[size] : null;\n        if (size > 0) {\n            if (heap) {\n                smallHeap[0] = new SmallSlot<byte[]>(\"heap-small\", new byte[] {(byte) seed, 1});\n                normalHeap[0] = new NormalSlot<byte[]>(\"heap-normal\", new byte[] {(byte) (seed + 2), 2});\n            }\n            if (direct) {\n                smallDirect[0] = new SmallSlot<ByteBuffer>(\"direct-small\", ByteBuffer.wrap(new byte[] {(byte) (seed + 1), 3}));\n                normalDirect[0] = new NormalSlot<ByteBuffer>(\"direct-normal\", ByteBuffer.wrap(new byte[] {(byte) (seed + 3), 4}));\n            }\n        }\n    }\n\n    public Access borrow(Lane<?> lane, int sizeIdx, Band band, int delta) {\n        Slot<?> selected = cache(lane, sizeIdx, band);\n        if (selected != null) { selected.touch(delta); }\n        return selected;\n    }\n    public Access find(Lane<?> lane, int sizeIdx, Band band) {\n        return cache(lane, sizeIdx, band);\n    }\n    public Access explicitHeap(int index) {\n        return Registry.<byte[]>cache(smallHeap, index);\n    }\n    public byte[] replaceHeap(int index, byte[] next) {\n        Slot<byte[]> selected = cache(smallHeap, index);\n        if (selected == null) { return null; }\n        byte[] previous = selected.value;\n        selected.value = next;\n        return previous;\n    }\n    public ByteBuffer replaceDirect(int index, ByteBuffer next) {\n        Slot<ByteBuffer> selected = cache(smallDirect, index);\n        if (selected == null) { return null; }\n        ByteBuffer previous = selected.value;\n        selected.value = next;\n        return previous;\n    }\n    private Slot<?> cache(Lane<?> lane, int sizeIdx, Band band) {\n        routes++;\n        switch (band) {\n        case NORMAL: return cacheForNormal(lane, sizeIdx);\n        case SMALL: return cacheForSmall(lane, sizeIdx);\n        default: throw new Error();\n        }\n    }\n    private Slot<?> cacheForSmall(Lane<?> lane, int sizeIdx) {\n        if (lane.isDirect()) {\n            return cache(smallDirect, sizeIdx);\n        }\n        return cache(smallHeap, sizeIdx);\n    }\n    private Slot<?> cacheForNormal(Lane<?> lane, int sizeIdx) {\n        int idx = sizeIdx - lane.subpages();\n        if (lane.isDirect()) {\n            return cache(normalDirect, idx);\n        }\n        return cache(normalHeap, idx);\n    }\n    private static <T> Slot<T> cache(Slot<T>[] cache, int sizeIdx) {\n        if (cache == null || sizeIdx > cache.length - 1) {\n            return null;\n        }\n        return cache[sizeIdx];\n    }\n    public String snapshot() { return \"routes=\" + routes; }\n\n    private abstract static class Slot<T> implements Access {\n        private final String label;\n        private T value;\n        private int touches;\n        private long credit;\n        Slot(String label, T value) { this.label = label; this.value = value; }\n        public String label() { return label + \"/\" + band(); }\n        public Object payload() { return value; }\n        public void touch(int delta) { touches++; credit += delta; }\n        public int touches() { return touches; }\n        public long credit() { return credit; }\n        abstract String band();\n    }\n    private static final class SmallSlot<T> extends Slot<T> {\n        SmallSlot(String label, T value) { super(label, value); }\n        String band() { return \"small\"; }\n    }\n    private static final class NormalSlot<T> extends Slot<T> {\n        NormalSlot(String label, T value) { super(label, value); }\n        String band() { return \"normal\"; }\n    }\n}\n",
		"campaign/cache/validation/Checks.java": "package campaign.cache.validation;\n\npublic final class Checks {\n    private Checks() { }\n    public static int requireNonNegative(int value) {\n        if (value < 0) { throw new IllegalArgumentException(\"size=\" + value); }\n        return value;\n    }\n}\n",
	}
}
func enumSwitch184Control() map[string]string {
	return map[string]string{
		"enumprobe/api/Lane.java":  "package enumprobe.api; public class Lane {public enum Band {SMALL,NORMAL}}\n",
		"enumprobe/use/Probe.java": "package enumprobe.use;\nimport enumprobe.api.Lane.Band;\npublic class Probe {\n static int NORMAL=99;\n static int selections;\n static Band select(Band band){selections++;return band;}\n static int colon(Band band){switch(select(band)){case NORMAL:return 7;case SMALL:return 3;default:return -1;}}\n static int arrow(Band band){switch(band){case NORMAL,SMALL -> {return 5;}default -> {return -1;}}}\n static int expression(Band band){return switch(band){case NORMAL -> 7;case SMALL -> 3;};}\n static String nullable(Band band){try{return Integer.toString(colon(band));}catch(NullPointerException expected){return \"NPE\";}}\n public static void main(String[] args){System.out.println(colon(Band.NORMAL)+\":\"+colon(Band.SMALL)+\":\"+arrow(Band.NORMAL)+\":\"+expression(Band.NORMAL)+\":\"+expression(Band.SMALL)+\":\"+nullable(null)+\":\"+selections);}\n}\n",
	}
}

func enumSwitch184ExpressionText(t *testing.T, expression ast.Expr) string {
	t.Helper()
	var text bytes.Buffer
	if err := printer.Fprint(&text, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	return text.String()
}

func enumSwitch184CheckMethod(t *testing.T, files map[string]string, caller, className, method string) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, files, caller)
	scope := ctx.currentFile.FindClassScope(className)
	if scope == nil {
		t.Fatal("fixture caller class missing")
	}
	ctx = classScopeCtx(scope, ctx)
	definitions := scope.FindMethod().ByOriginalName(method)
	selected, matches := -1, 0
	for index, definition := range definitions {
		var containsSwitch func(*sitter.Node) bool
		containsSwitch = func(node *sitter.Node) bool {
			if node == nil {
				return false
			}
			if node.Type() == "switch_statement" || node.Type() == "switch_expression" {
				return true
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if containsSwitch(child) {
					return true
				}
			}
			return false
		}
		if containsSwitch(definition.DeclarationNode) {
			selected, matches = index, matches+1
		}
	}
	if matches != 1 {
		t.Fatal("fixture unique switch-containing method missing")
	}
	ctx.localScope = definitions[selected]
	ctx.importAliases = map[string]string{}
	ctx.usedImports = map[string]bool{}
	source := ctx.currentFile.Source
	enum := resolveClassScopeByQualifiedName(ctx, "Band")
	if enum == nil || !enum.IsEnum || enum.Class == nil {
		t.Fatal("fixture exact imported enum owner missing")
	}
	checked := 0
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "switch_block_statement_group" || node.Type() == "switch_rule" {
			var labels []*sitter.Node
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if child.Type() == "switch_label" {
					labels = append(labels, nodeutil.SemanticNamedChildrenOf(child)...)
				}
			}
			var cases []ast.Expr
			if node.Type() == "switch_rule" {
				cases, _, _ = splitSwitchRule(node, source, ctx, nil)
			} else {
				cases, _, _ = splitSwitchGroup(node, source, ctx, nil)
			}
			if len(cases) != len(labels) {
				t.Fatalf("fixture switch label count got%d want%d", len(cases), len(labels))
			}
			for index, label := range labels {
				if label.Type() != "identifier" {
					t.Fatal("fixture enum label is not identifier")
				}
				name := label.Content(source)
				var field *fieldResolution
				for _, constant := range enum.EnumConstants {
					if constant.Name == name && constant.Field != nil && constant.Field.OriginalName == name {
						field = &fieldResolution{def: constant.Field, owner: enum}
					}
				}
				if field == nil {
					t.Fatal("fixture exact enum declaration field missing")
				}
				expected := enumSwitch184ExpressionText(t, staticFieldStorageExpr(&staticFieldAccess{resolution: field}, ctx))
				actual := enumSwitch184ExpressionText(t, cases[index])
				if actual != expected {
					t.Errorf("enum contextual switch label lacks exact declaration storage: method=%s label=%s got=%s want=%s", method, name, actual, expected)
				}
				checked++
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if checked != 2 {
		t.Fatalf("fixture must exercise both actualenum constants; checked=%d", checked)
	}
}

func TestEnumSwitch184ActualOriginalColonDeclarations(t *testing.T) {
	enumSwitch184CheckMethod(t, enumSwitch184Original5(), "campaign/cache/store/Registry.java", "Registry", "cache")
}

func TestEnumSwitch184ArrowAndExpressionDeclarations(t *testing.T) {
	for _, method := range []string{"colon", "arrow", "expression"} {
		t.Run(method, func(t *testing.T) {
			enumSwitch184CheckMethod(t, enumSwitch184Control(), "enumprobe/use/Probe.java", "Probe", method)
		})
	}
}

func TestEnumSwitch184ContextualLabelsJDK21(t *testing.T) {
	files := enumSwitch184Control()
	files["pom.xml"] = "<project><groupId>enumprobe</groupId><artifactId>contextual-labels</artifactId><version>1</version></project>\n"
	project := map[string]string{"pom.xml": files["pom.xml"]}
	for name, text := range enumSwitch184Control() {
		project["src/main/java/"+name] = text
	}
	runCampaignCompilerStrictProjectObservations(t, project, "enumprobe.use.Probe", []campaignCompilerProjectObservation{{name: "first", stdout: "7:3:5:7:3:NPE:3\n"}, {name: "second", stdout: "7:3:5:7:3:NPE:3\n"}, {name: "third", stdout: "7:3:5:7:3:NPE:3\n"}})
}

func enumSwitch184LabelFixture(t *testing.T) (*sitter.Node, Ctx) {
	t.Helper()
	files := enumSwitch184Control()
	ctx := ordinaryDemandTestContext(t, files, "enumprobe/use/Probe.java")
	scope := ctx.currentFile.FindClassScope("Probe")
	ctx = classScopeCtx(scope, ctx)
	definitions := scope.FindMethod().ByOriginalName("colon")
	if len(definitions) != 1 {
		t.Fatal("fixture colon method missing")
	}
	ctx.localScope = definitions[0]
	ctx.importAliases = map[string]string{}
	ctx.usedImports = map[string]bool{}
	var label *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || label != nil {
			return
		}
		if node.Type() == "identifier" && node.Content(ctx.currentFile.Source) == "NORMAL" && node.Parent() != nil && node.Parent().Type() == "switch_label" {
			label = node
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(definitions[0].DeclarationNode)
	if label == nil {
		t.Fatal("fixture NORMAL label missing")
	}
	return label, ctx
}

func TestEnumSwitch184LiveFieldAndAliasOwnership(t *testing.T) {
	label, ctx := enumSwitch184LabelFixture(t)
	owner := resolveClassScopeByQualifiedName(ctx, "Band")
	if owner == nil || !owner.IsEnum {
		t.Fatal("fixture enum owner missing")
	}
	for index := range owner.EnumConstants {
		constant := &owner.EnumConstants[index]
		if constant.Name != "NORMAL" {
			continue
		}
		field := constant.Field
		for _, row := range []struct{ name, alias string }{{"RenamedConstant", "ownedAPI"}, {"LaterConstant", "laterAPI"}} {
			field.Name = row.name
			ctx.importAliases["enumprobe.api"] = row.alias
			ctx.usedImports = map[string]bool{}
			got := enumSwitch184ExpressionText(t, parseSwitchLabelExpression(label, ctx.currentFile.Source, ctx, nil))
			expected := enumSwitch184ExpressionText(t, staticFieldStorageExpr(&staticFieldAccess{resolution: &fieldResolution{def: field, owner: owner}}, ctx))
			if got != expected || !ctx.usedImports["enumprobe.api"] {
				t.Fatalf("live declaration/alias binding lost: got=%s want=%s used=%v", got, expected, ctx.usedImports)
			}
		}
		return
	}
	t.Fatal("fixture exact constant missing")
}

func TestEnumSwitch184UncertainProofRetainsOrdinaryExpression(t *testing.T) {
	for _, name := range []string{"non_enum_owner", "missing_constant_field", "nonconstant_field", "wrong_original_name", "wrong_declaration_kind", "edited_constant_source", "ordinary_identifier", "unknown_label"} {
		t.Run(name, func(t *testing.T) {
			label, ctx := enumSwitch184LabelFixture(t)
			owner := resolveClassScopeByQualifiedName(ctx, "Band")
			if owner == nil || !owner.IsEnum {
				t.Fatal("fixture enum owner missing")
			}
			constantIndex := -1
			for i := range owner.EnumConstants {
				if owner.EnumConstants[i].Name == "NORMAL" {
					constantIndex = i
				}
			}
			if constantIndex < 0 {
				t.Fatal("fixture constant missing")
			}
			constant := &owner.EnumConstants[constantIndex]
			switch name {
			case "non_enum_owner":
				owner.IsEnum = false
			case "missing_constant_field":
				constant.Field = nil
			case "nonconstant_field":
				constant.Field.IsFinal = false
			case "wrong_original_name":
				constant.Field.OriginalName = "OTHER"
			case "wrong_declaration_kind":
				constant.Field.DeclarationNode = ctx.localScope.DeclarationNode
			case "edited_constant_source":
				file := findFileScopeForClassScope(owner, ctx)
				if file == nil {
					t.Fatal("fixture declaring file missing")
				}
				fieldName := constant.Field.DeclarationNode.ChildByFieldName("name")
				copy(file.Source[fieldName.StartByte():fieldName.EndByte()], []byte("NORMAX"))
			case "ordinary_identifier":
				label = ctx.localScope.DeclarationNode.ChildByFieldName("name")
			case "unknown_label":
				copy(ctx.currentFile.Source[label.StartByte():label.EndByte()], []byte("ABSENT"))
			}
			ordinary, oldPanic := enumSwitch184ObserveExpression(t, func() ast.Expr { return ParseExpr(label, ctx.currentFile.Source, ctx) })
			before := len(Diagnostics())
			got, newPanic := enumSwitch184ObserveExpression(t, func() ast.Expr { return parseSwitchLabelExpression(label, ctx.currentFile.Source, ctx, nil) })
			if reflect.TypeOf(oldPanic) != reflect.TypeOf(newPanic) {
				t.Fatalf("uncertain proof changed panic type: old=%T new=%T", oldPanic, newPanic)
			}
			if oldPanic != nil {
				if !reflect.TypeOf(oldPanic).Comparable() {
					t.Fatal("oracle panic must support exact identity comparison")
				}
				if oldPanic != newPanic {
					t.Fatalf("uncertain proof changed original panic identity: old=%v new=%v", oldPanic, newPanic)
				}
			}
			if got != ordinary || len(Diagnostics()) != before {
				t.Fatalf("uncertain proof changed ordinary path: got=%s want=%s diagnostics=%v", got, ordinary, Diagnostics())
			}
		})
	}
}

func TestEnumSwitch184ForeignTreeAndMissingOwnershipFallback(t *testing.T) {
	for _, name := range []string{"foreign_tree", "missing_context", "missing_parse_certificate", "source_mismatch"} {
		t.Run(name, func(t *testing.T) {
			label, ctx := enumSwitch184LabelFixture(t)
			source := ctx.currentFile.Source
			var keepAlive any
			switch name {
			case "foreign_tree":
				foreign := &parsing.SourceFile{Name: "Foreign.java", Source: append([]byte(nil), source...)}
				if err := foreign.ParseAST(); err != nil {
					t.Fatal(err)
				}
				keepAlive = foreign
				var found *sitter.Node
				var walk func(*sitter.Node)
				walk = func(node *sitter.Node) {
					if node == nil || found != nil {
						return
					}
					if node.Type() == "identifier" && node.Content(source) == "NORMAL" && node.Parent() != nil && node.Parent().Type() == "switch_label" {
						found = node
						return
					}
					for _, child := range nodeutil.NamedChildrenOf(node) {
						walk(child)
					}
				}
				walk(foreign.Ast)
				if found == nil {
					t.Fatal("fixture foreign label missing")
				}
				label = found
			case "missing_context":
				ctx.currentFile = nil
			case "missing_parse_certificate":
				ctx.currentFile = &symbol.FileScope{Source: source, Package: ctx.currentFile.Package, Imports: ctx.currentFile.Imports}
			case "source_mismatch":
				source = append([]byte(nil), source...)
				source[0] = 'q'
			}
			ordinary := enumSwitch184ExpressionText(t, ParseExpr(label, source, ctx))
			before := len(Diagnostics())
			actual := enumSwitch184ExpressionText(t, parseSwitchLabelExpression(label, source, ctx, nil))
			if actual != ordinary || len(Diagnostics()) != before || ctx.usedImports["enumprobe.api"] {
				t.Fatalf("foreign/missing certificate acquired enum label authority: got=%s want=%s imports=%v", actual, ordinary, ctx.usedImports)
			}
			runtime.KeepAlive(keepAlive)
		})
	}
}

func enumSwitch184ObserveExpression(t *testing.T, emit func() ast.Expr) (text string, failure any) {
	t.Helper()
	defer func() { failure = recover() }()
	text = enumSwitch184ExpressionText(t, emit())
	return text, nil
}
