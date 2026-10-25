package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"os"
	"testing"
)

func sourceStorageInterfaceFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/source_interface_storage/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func TestGenericFamilySourceInterfaceStorageCompleteDependency(t *testing.T) {
	h := setupParseHelper(t, sourceStorageInterfaceFixture(t))
	root := h.File.Symbols.FindClassScope("Carrier")
	p, err := planGenericFamily(root, h.Ctx)
	if err != nil || p == nil {
		t.Fatalf("whole source interface dependency refused: %v", err)
	}
	for _, name := range []string{"Carrier", "Deferred", "Source", "ValueSource"} {
		c := h.File.Symbols.FindClassScope(name)
		if c == nil {
			t.Fatal("missing declaration", name)
		}
		if _, ok := p.members[c]; !ok {
			t.Fatal("missing whole connected source declaration", name)
		}
		for _, v := range c.TypeParameters {
			if _, ok := p.binders[v.Declaration]; !ok {
				t.Fatal("dependency omitted actual declaration binder", name)
			}
		}
	}
}
func TestGenericFamilySourceInterfaceStorageRejectsNamedPartialImplementor(t *testing.T) {
	s := sourceStorageInterfaceFixture(t) + `class Bad<B> implements Source<B>{B[] unsafe;public B construct(){return null;}}`
	h := setupParseHelper(t, s)
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("unsupported named implementor did not close entire dependent family")
	}
}
func TestGenericFamilySourceInterfaceStorageRejectsAnonymousPartialImplementor(t *testing.T) {
	s := `interface Source<A>{A construct();}abstract class Carrier<A>{abstract A read();}class Deferred<A> extends Carrier<A>{Source<A> source;A read(){return source.construct();}Source<A> hidden(){return new Source<A>(){A[] unsafe;public A construct(){return null;}};}}`
	h := setupParseHelper(t, s)
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("unsupported anonymous implementor escaped dependency audit")
	}
}
func TestGenericFamilySourceInterfaceStorageRejectsLocalPartialImplementor(t *testing.T) {
	s := `interface Source<A>{A construct();}abstract class Carrier<A>{abstract A read();}class Deferred<A> extends Carrier<A>{Source<A> source;A read(){return source.construct();}Source<A> hidden(){class Bad implements Source<A>{A[] unsafe;public A construct(){return null;}}return new Bad();}}`
	h := setupParseHelper(t, s)
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("unsupported local implementor escaped dependency audit")
	}
}
func TestGenericFamilySourceInterfaceStorageClosesMultipleAndCyclicDependencies(t *testing.T) {
	s := `interface Origin<A>{A get();}interface Twin<B>{B get();}class Both<C> implements Origin<C>,Twin<C>{C value;Sibling<C> sibling;public C get(){return value;}}class Sibling<D>{Origin<D> source;}abstract class Carrier<E>{abstract E read();}class Deferred<E> extends Carrier<E>{Origin<E> source;E read(){return source.get();}}`
	h := setupParseHelper(t, s)
	p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
	if p == nil || err != nil {
		t.Fatalf("complete cyclic source storage refused: %v", err)
	}
	for _, name := range []string{"Origin", "Twin", "Both", "Sibling"} {
		if _, ok := p.members[h.File.Symbols.FindClassScope(name)]; !ok {
			t.Fatal("incomplete connected storage component", name)
		}
	}
	h = setupParseHelper(t, s+`class Broken<V> implements Twin<V>{V[] unsafe;public V get(){return null;}}`)
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("second interface implementor escaped complete closure")
	}
}

// Later's implementors are outside the first dependency's hierarchy; only a
// subsequent storage-seed round can inventory them.
func TestGenericFamilySourceInterfaceStorageLaterRoundClosureAndFiniteWork(t *testing.T) {
	const prefix = `interface First<A>{A get();}interface Later<B>{B get();}class FirstImpl<C> implements First<C>{Later<C> later;public C get(){return later.get();}}class LaterImpl<D> implements Later<D>{D value;First<D> back;public D get(){return value;}}abstract class Carrier<E>{abstract E read();}class Deferred<E> extends Carrier<E>{First<E> source;E read(){return source.get();}`
	h := setupParseHelper(t, prefix+`}`)
	inventory := newGenericFamilyInventory(h.Ctx)
	p, err := planGenericFamilyWithInventory(h.File.Symbols.FindClassScope("Carrier"), h.Ctx, inventory)
	if p == nil || err != nil {
		t.Fatalf("later-seed cycle refused: %v", err)
	}
	for _, name := range []string{"First", "FirstImpl", "Later", "LaterImpl"} {
		if _, ok := p.members[h.File.Symbols.FindClassScope(name)]; !ok {
			t.Fatal("later dependency omitted", name)
		}
	}
	if inventory.storageAuditRounds != 3 || inventory.storageScopesAudited != 12 || inventory.storageSourceEventScans != 1 || inventory.storageComponentScans != 1 {
		t.Fatalf("restart work drift: rounds=%d scopes=%d sourceScans=%d", inventory.storageAuditRounds, inventory.storageScopesAudited, inventory.storageSourceEventScans)
	}
	t.Logf("two interface storage dependencies with backedge: completeAuditRounds=%d declarationAudits=%d structuralSourceInventories=%d componentInventories=%d", inventory.storageAuditRounds, inventory.storageScopesAudited, inventory.storageSourceEventScans, inventory.storageComponentScans)
	for _, test := range []struct{ name, tail, extra string }{
		{"named", `}`, `class Bad<X> implements Later<X>{X[] unsafe;public X get(){return null;}}`},
		{"anonymous", `Later<E> bad(){return new Later<E>(){E[] unsafe;public E get(){return null;}};}}`, ``},
		{"local", `Later<E> bad(){class Bad implements Later<E>{E[] unsafe;public E get(){return null;}}return new Bad();}}`, ``},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := setupParseHelper(t, prefix+test.tail+test.extra)
			if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("later-seed unsupported implementor escaped complete audit")
			}
		})
	}
}

func TestGenericFamilySourceInterfaceStorageOwnsOccurrenceAndRegisteredDeclaration(t *testing.T) {
	source := sourceStorageInterfaceFixture(t)
	h := setupParseHelper(t, source)
	target := h.File.Symbols.FindClassScope("Source")
	findUse := func(root *sitter.Node, source []byte) *sitter.Node {
		var found *sitter.Node
		var walk func(*sitter.Node)
		walk = func(n *sitter.Node) {
			if n == nil || found != nil {
				return
			}
			if n.Type() == "generic_type" && n.Content(source) == "Source<A>" {
				found = n
				return
			}
			for _, child := range nodeutil.NamedChildrenOf(n) {
				walk(child)
			}
		}
		walk(root)
		return found
	}
	use := findUse(h.File.Symbols.FindClassScope("Deferred").Class.DeclarationNode, h.File.Symbols.Source)
	if use == nil {
		t.Fatal("missing original field type")
	}
	currentGraph := symbol.GlobalScope
	inventory := newGenericFamilyInventory(h.Ctx)
	foreign := setupParseHelper(t, source)
	foreignUse := findUse(foreign.File.Symbols.FindClassScope("Deferred").Class.DeclarationNode, foreign.File.Symbols.Source)
	symbol.GlobalScope = currentGraph
	if foreignUse == nil {
		t.Fatal("missing identical foreign type")
	}
	plan := &genericFamilyPlan{storageInventory: inventory}
	if plan.queueSourceInterfaceStorage(target, "Source<A>", h.Ctx, foreignUse) {
		t.Fatal("foreign tree with equal source span donated storage ownership")
	}
	copy := *target
	if plan.queueSourceInterfaceStorage(&copy, "Source<A>", h.Ctx, use) {
		t.Fatal("unregistered copied scope donated storage ownership")
	}
	if (&genericFamilyPlan{}).queueSourceInterfaceStorage(target, "Source<A>", h.Ctx, use) {
		t.Fatal("fabricated binder-only plan acquired a storage capability")
	}
	if !plan.queueSourceInterfaceStorage(target, "Source<A>", h.Ctx, use) {
		t.Fatal("actual registered source occurrence lost ownership")
	}
}

func TestGenericFamilySourceInterfaceStorageKeepsArgumentRecursionAndArrayRefusals(t *testing.T) {
	for _, typ := range []string{"Source<java.util.List<E>>", "Source<E>[]", "Source<E[]>", "ValueSource<E>[]"} {
		t.Run(typ, func(t *testing.T) {
			source := `interface Source<A>{A construct();}class ValueSource<A> implements Source<A>{A value;public A construct(){return value;}}abstract class Carrier<E>{abstract E read();}class Deferred<E> extends Carrier<E>{Source<E> source;` + typ + ` unsafe;E read(){return source.construct();}}`
			h := setupParseHelper(t, source)
			if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("source interface anchor donated unsupported argument or array layout")
			}
		})
	}
}
