package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func sourceClassStorageFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/date_source_class_storage/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func sourceClassStorageScope(t *testing.T, name string) *symbol.ClassScope {
	t.Helper()
	var found *symbol.ClassScope
	for _, s := range allSourceClassScopes() {
		if s.Class != nil && s.Class.OriginalName == name {
			if found != nil {
				t.Fatal("duplicate declaration", name)
			}
			found = s
		}
	}
	if found == nil {
		t.Fatal("missing declaration", name)
	}
	return found
}
func TestGenericFamilySourceClassStorageDateComplete(t *testing.T) {
	h := setupParseHelper(t, sourceClassStorageFixture(t))
	root := sourceClassStorageScope(t, "Carrier")
	inventory := newGenericFamilyInventory(h.Ctx)
	p, err := planGenericFamilyWithInventory(root, h.Ctx, inventory)
	if p == nil || err != nil {
		t.Fatalf("complete Date class storage component refused: %v", err)
	}
	for _, name := range []string{"Carrier", "Deferred", "Kind", "ValueKind"} {
		c := sourceClassStorageScope(t, name)
		if _, ok := p.members[c]; !ok {
			t.Fatal("missing complete class dependency", name)
		}
		params := c.OwnTypeParameters()
		if len(params) != 1 {
			t.Fatal("missing original own binder", name)
		}
		rep, ok := p.representations[params[0].Declaration]
		want := "DateValue"
		if name == "Carrier" {
			want = "any"
		}
		if !ok || rep.goName != want {
			t.Fatalf("declaration %s physical representation %#v, want %s", name, rep, want)
		}
	}
	factory := sourceClassStorageScope(t, "Factory")
	if len(factory.Methods) != 1 || !canonicalGenericMethodSignature(factory.Methods[0], classScopeCtx(factory, h.Ctx)) {
		t.Fatal("factory callable refused despite complete class component")
	}
	if inventory.storageAuditRounds != 2 || inventory.storageComponentScans != 1 {
		t.Fatalf("class dependency restart work rounds=%d scans=%d", inventory.storageAuditRounds, inventory.storageComponentScans)
	}
	t.Logf("class component completeAuditRounds=%d declarationAudits=%d structuralComponents=%d", inventory.storageAuditRounds, inventory.storageScopesAudited, inventory.storageComponentScans)
}

const sourceClassStorageNestedPrefix = `import java.util.Date;abstract class Carrier<T>{abstract T read();}class Shell<T>{static abstract class Kind<T extends Date>{Class<T> token;Deferred<T> back;Kind(Class<T> token){this.token=token;}abstract T read();}}class ValueKind<T extends Date> extends Shell.Kind<T>{T value;ValueKind(Class<T> token,T value){super(token);this.value=value;}T read(){return value;}}class Deferred<T extends Date> extends Carrier<T>{Shell.Kind<T> kind;T read(){return kind.read();}`

func TestGenericFamilySourceClassStorageStaticNestedCycle(t *testing.T) {
	h := setupParseHelper(t, sourceClassStorageNestedPrefix+`}`)
	p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx)
	if p == nil || err != nil {
		t.Fatalf("static nested bounded component cycle refused: %v", err)
	}
	shell := sourceClassStorageScope(t, "Shell")
	kind := sourceClassStorageScope(t, "Kind")
	if _, ok := p.members[shell]; ok {
		t.Fatal("static nesting invented an outer layout edge")
	}
	if shell.OwnTypeParameters()[0].Declaration == kind.OwnTypeParameters()[0].Declaration {
		t.Fatal("same-spelling lexical binders collapsed")
	}
	for _, name := range []string{"Carrier", "Deferred", "Kind", "ValueKind"} {
		if _, ok := p.members[sourceClassStorageScope(t, name)]; !ok {
			t.Fatal("cycle omitted declaration", name)
		}
	}
	if p.representations[kind.OwnTypeParameters()[0].Declaration].goName != "DateValue" {
		t.Fatal("nested own Date binder lost native physical representation")
	}
}
func TestGenericFamilySourceClassStorageRejectsWholeComponent(t *testing.T) {
	for _, c := range []struct{ name, tail, extra string }{
		{"named", `}`, `class Bad<X extends Date> extends Shell.Kind<X>{java.util.List<X> unsafe;Bad(Class<X> token){super(token);}X read(){return null;}}`},
		{"anonymous", `Shell.Kind<T> bad(Class<T> token){return new Shell.Kind<T>(token){java.util.List<T> unsafe;T read(){return null;}};}}`, ``},
		{"local_superclass", `Shell.Kind<T> bad(Class<T> token){class Local extends Shell.Kind<T>{Local(){super(token);}T read(){return null;}}return new Local();}}`, ``},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := setupParseHelper(t, sourceClassStorageNestedPrefix+c.tail+c.extra)
			if p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("unsupported later class implementor escaped complete audit")
			}
		})
	}
}
func TestGenericFamilySourceClassStorageCannotDonateNativeLayout(t *testing.T) {
	t.Run("existing_canonical_Map_reference_contract", func(t *testing.T) {
		h := setupParseHelper(t, sourceClassStorageNestedPrefix+`java.util.Map<String,T> supported;}`)
		if !mapReferenceArgumentIndependent("java.util.Map<String,T>", classScopeCtx(sourceClassStorageScope(t, "Deferred"), h.Ctx)) {
			t.Fatal("existing canonical Map reference certificate missing")
		}
		if p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx); p == nil || err != nil {
			t.Fatalf("existing Map reference ABI refused: %v", err)
		}
	})

	for _, typ := range []string{"java.util.List<T>", "java.util.HashMap<String,T>"} {
		t.Run(typ, func(t *testing.T) {
			h := setupParseHelper(t, sourceClassStorageNestedPrefix+typ+` unsafe;}`)
			if p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("source class anchor donated native generic container ABI")
			}
		})
	}
	for _, bound := range []string{"java.net.URI", "java.sql.Date"} {
		t.Run(bound, func(t *testing.T) {
			s := `abstract class Carrier<T>{abstract T read();}abstract class Kind<T extends ` + bound + `>{abstract T read();}class Deferred<T extends ` + bound + `> extends Carrier<T>{Kind<T> kind;T read(){return kind.read();}}`
			h := setupParseHelper(t, s)
			if p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("component anchor donated unknown native bound representation")
			}
		})
	}
}

func TestGenericFamilySourceClassStorageOwnsOccurrenceAndDeclaration(t *testing.T) {
	source := sourceClassStorageFixture(t)
	h := setupParseHelper(t, source)
	target := sourceClassStorageScope(t, "Kind")
	find := func(root *sitter.Node, source []byte) *sitter.Node {
		var use *sitter.Node
		var walk func(*sitter.Node)
		walk = func(n *sitter.Node) {
			if n == nil || use != nil {
				return
			}
			if n.Type() == "generic_type" && n.Content(source) == "Kind<T>" {
				use = n
				return
			}
			for _, c := range nodeutil.NamedChildrenOf(n) {
				walk(c)
			}
		}
		walk(root)
		return use
	}
	use := find(sourceClassStorageScope(t, "Deferred").Class.DeclarationNode, h.File.Symbols.Source)
	if use == nil {
		t.Fatal("missing current class storage occurrence")
	}
	graph := symbol.GlobalScope
	inventory := newGenericFamilyInventory(h.Ctx)
	foreign := setupParseHelper(t, source)
	other := find(sourceClassStorageScope(t, "Deferred").Class.DeclarationNode, foreign.File.Symbols.Source)
	symbol.GlobalScope = graph
	if other == nil {
		t.Fatal("missing equal-span foreign occurrence")
	}
	p := &genericFamilyPlan{storageInventory: inventory}
	if p.queueSourceComponentStorage(target, "Kind<T>", h.Ctx, other) {
		t.Fatal("foreign AST donated class storage ownership")
	}
	copy := *target
	if p.queueSourceComponentStorage(&copy, "Kind<T>", h.Ctx, use) {
		t.Fatal("copied unregistered class donated ownership")
	}
	if !p.queueSourceComponentStorage(target, "Kind<T>", h.Ctx, use) {
		t.Fatal("registered current class occurrence refused")
	}
}
func TestGenericFamilySourceClassStorageAnonymousBoundedComponent(t *testing.T) {
	source := sourceClassStorageNestedPrefix + `Shell.Kind<T> anonymous(Class<T> token){return new Shell.Kind<T>(token){T value;T read(){return value;}};}}`
	h := setupParseHelper(t, source)
	p, err := planGenericFamily(sourceClassStorageScope(t, "Carrier"), h.Ctx)
	if p == nil || err != nil || len(p.anonymous) != 1 {
		t.Fatalf("bounded class anonymous implementation not fully inventoried: plan=%v error=%v", p, err)
	}
}

func TestGenericFamilySourceClassStorageAnonymousReceiversCanonical(t *testing.T) {
	b, err := os.ReadFile("testdata/date_source_class_storage_nested/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	text := renderGoFileFromJava(t, string(b))
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", text, 0)
	if err != nil {
		t.Fatal(err)
	}
	aliases := map[string]string{}
	for _, d := range file.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range g.Specs {
			typ, ok := s.(*ast.TypeSpec)
			if !ok || typ.Assign == token.NoPos || typ.TypeParams == nil {
				continue
			}
			if raw, ok := typ.Type.(*ast.Ident); ok {
				aliases[typ.Name.Name] = raw.Name
			}
		}
	}
	if len(aliases) == 0 {
		t.Fatal("complete source component emitted no canonical aliases")
	}
	receivers := 0
	for _, d := range file.Decls {
		m, ok := d.(*ast.FuncDecl)
		if !ok || m.Recv == nil {
			continue
		}
		p, ok := m.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		var base ast.Expr
		switch r := p.X.(type) {
		case *ast.IndexExpr:
			base = r.X
		case *ast.IndexListExpr:
			base = r.X
		}
		if name, ok := base.(*ast.Ident); ok && aliases[name.Name] != "" {
			t.Fatal("method emitted on generic alias", m.Name.Name, name.Name)
		}
		raw, ok := p.X.(*ast.Ident)
		if !ok {
			continue
		}
		owned := false
		for _, name := range aliases {
			owned = owned || name == raw.Name
		}
		if !owned {
			continue
		}
		receivers++
		ast.Inspect(m.Type, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "M" {
				t.Fatal("hoisted receiver signature retained enclosing method binder", m.Name.Name)
			}
			return true
		})
	}
	if receivers == 0 {
		t.Fatal("no canonical receiver methods inspected")
	}
}
