package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each fixture and oracle is the full frozen, independently JDK21-validated source.
func canonicalClassPhysicalProject(t *testing.T, fixture, main string, names []string) {
	t.Helper()
	root := filepath.Join("testdata", "canonical_class_physical129", fixture)
	files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>classproof</groupId><artifactId>class-physical</artifactId><version>1</version></project>`}
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Join("src/main/java", name)] = string(content)
	}
	want, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, main, string(want))
}

func TestCanonicalClassPhysicalCanonicalClassReflectionOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "CanonicalClassReflectionOracle", "classproof.CanonicalClassReflectionOracle", []string{"CanonicalClassReflectionOracle.java"})
}

func TestCanonicalClassPhysicalCanonicalClassStorageOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "CanonicalClassStorageOracle", "classproof.CanonicalClassStorageOracle", []string{"CanonicalClassStorageOracle.java"})
}

func TestCanonicalClassPhysicalClassOwnerGuardsOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "ClassOwnerGuardsOracle", "guards.ClassOwnerGuardsOracle", []string{"ClassOwnerGuardsOracle.java", "Class.java"})
}

func TestCanonicalClassPhysicalLexicalClassOccurrenceOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "LexicalClassOccurrenceOracle", "lexical.LexicalClassOccurrenceOracle", []string{"LexicalClassOccurrenceOracle.java"})
}

func TestCanonicalClassPhysicalNamespaceVariableOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "NamespaceVariableOracle", "namespace.NamespaceVariableOracle", []string{"NamespaceVariableOracle.java"})
}

func TestCanonicalClassPhysicalSourceTypePrefixOracle(t *testing.T) {
	canonicalClassPhysicalProject(t, "SourceTypePrefixOracle", "namespace.SourceTypePrefixOracle", []string{"SourceTypePrefixOracle.java"})
}

func TestCanonicalClassPhysicalFamilyStoragePlan(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"proof/Slot.java": `package proof;public class Slot<T>{Class<? super T> raw;Class<T> exact;Class<T>[] array;Class<T> read(){return exact;}void set(Class<T> next){exact=next;}}`})
	owner := findQualifiedSourceClass("proof.Slot")
	if _, err := planGenericFamily(owner, ctx); err != nil {
		t.Fatalf("canonical Class service storage rejected: %v", err)
	}
}

func TestCanonicalClassPhysicalFamilyDriver(t *testing.T) {
	canonicalClassPhysicalProject(t, "ClassFamilyCanonicalDriver", "classproof.ClassFamilyCanonicalDriver", []string{"CanonicalClassStorageOracle.java", "ClassFamilyCanonicalDriver.java"})
}

// Use the exact frozen JDK source nodes, including both canonical and shadowed
// occurrences in one method. These assertions concern new service eligibility;
// refusing it does not refuse the ordinary source-family path.
func TestCanonicalClassPhysicalOccurrenceOwnership(t *testing.T) {
	for _, test := range []struct {
		fixture, owner string
		names          map[string]bool
	}{
		{"LexicalClassOccurrenceOracle", "lexical.LexicalClassOccurrenceOracle", map[string]bool{"memberBeforeLocal": true, "before": true, "inside": false, "afterBlock": true, "sibling": false, "local": false, "nested": false, "captured": false, "qualified": true, "canonical": true}},
		{"NamespaceVariableOracle", "namespace.NamespaceVariableOracle", map[string]bool{"field": true, "result": true, "local": true, "token": true}},
		{"SourceTypePrefixOracle", "namespace.SourceTypePrefixOracle", map[string]bool{"source": false}},
		{"ClassOwnerGuardsOracle", "guards.ClassOwnerGuardsOracle", map[string]bool{"source": false, "other": false, "local": false}},
	} {
		t.Run(test.fixture, func(t *testing.T) {
			bytes, err := os.ReadFile(filepath.Join("testdata", "canonical_class_physical129", test.fixture, test.fixture+".java"))
			if err != nil {
				t.Fatal(err)
			}
			sources := map[string]string{test.fixture + ".java": string(bytes)}
			if test.fixture == "ClassOwnerGuardsOracle" {
				foreign, err := os.ReadFile(filepath.Join("testdata", "canonical_class_physical129", test.fixture, "Class.java"))
				if err != nil {
					t.Fatal(err)
				}
				sources["foreign/Class.java"] = string(foreign)
			}
			ctx := sourceGenericViewDemandTestContext(t, sources)
			owner := findQualifiedSourceClass(test.owner)
			ctx = classScopeCtx(owner, ctx)
			found := map[string]int{}
			var walk func(*sitter.Node, Ctx)
			walk = func(node *sitter.Node, lexical Ctx) {
				if node == nil {
					return
				}
				if node.Type() == "method_declaration" || node.Type() == "constructor_declaration" {
					lexical = lexical.Clone()
					lexical.localScope = nil
					for _, method := range owner.Methods {
						if sameSourceNode(method.DeclarationNode, node) {
							lexical.localScope = method
							break
						}
					}
					if lexical.localScope == nil && node.Type() == "method_declaration" {
						lexical.localScope = scopeForAnonymousMethod(node, synthAnonClassMethodDefinition(node, bytes, false), bytes)
					}
				}
				if node.Type() == "variable_declarator" {
					name := node.ChildByFieldName("name").Content(bytes)
					if want, ok := test.names[name]; ok {
						declaration := node.Parent()
						typ := declaration.ChildByFieldName("type")
						if typ != nil && strings.Contains(typ.Content(bytes), "Class") {
							got := canonicalClassArgumentIndependent(typ.Content(bytes), lexical, typ)
							if got != want {
								t.Errorf("%s @%d: Class service=%t want %t (type %s)", name, typ.StartByte(), got, want, typ.Content(bytes))
							}
							found[name]++
							if canonicalClassArgumentIndependent(typ.Content(bytes), lexical) {
								t.Errorf("%s admitted without an exact source occurrence", name)
							}
						}
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					walk(child, classDeclarationChildCtx(node, child, lexical))
				}
			}
			walk(owner.Class.DeclarationNode, ctx)
			for name := range test.names {
				if found[name] == 0 {
					t.Errorf("missing exact JDK type-use control %s", name)
				}
			}
		})
	}
}

func TestCanonicalClassPhysicalKeepsNestedStorageRejections(t *testing.T) {
	for _, body := range []string{`java.lang.Class<java.util.Collection<T>> hidden;`, `T[] ownedArray;`, `java.util.Date date;java.util.Collection<T> collection;`, `java.util.Map<String,T> map;`} {
		t.Run(body, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{"proof/Reject.java": `package proof;public class Reject<T>{` + body + `}`})
			if body != `java.util.Map<String,T> map;` {
				if _, err := planGenericFamily(findQualifiedSourceClass("proof.Reject"), ctx); err == nil {
					t.Fatal("Class service widened unrelated owned storage")
				}
				return
			}
			owner := findQualifiedSourceClass("proof.Reject")
			if owner == nil || len(owner.TypeParameters) != 1 || len(owner.Fields) != 1 {
				t.Fatal("original Map field or owned binder lost")
			}
			ctx = classScopeCtx(owner, ctx)
			field := owner.Fields[0]
			parameter := owner.TypeParameters[0]
			if field.OriginalName != "map" || field.OriginalType != "java.util.Map<String,T>" || parameter.Name != "T" || parameter.Declaration == nil || field.TypeParameterBindings["T"] != parameter.Declaration {
				t.Fatal("Map field lost its exact declaration-owned T")
			}
			var occurrence *sitter.Node
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil {
					return
				}
				if node.Type() == "field_declaration" {
					occurrence = node.ChildByFieldName("type")
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					walk(child)
				}
			}
			walk(owner.Class.DeclarationNode)
			if occurrence == nil || occurrence.Content(ctx.currentFile.Source) != field.OriginalType {
				t.Fatal("missing original Map field type occurrence")
			}
			if canonicalClassArgumentIndependent(field.OriginalType, ctx, occurrence) {
				t.Fatal("Map borrowed canonical Class eligibility")
			}
			if canonicalMapReferenceOwner(field.OriginalType, ctx) != "java.util.Map" || !mapReferenceArgumentIndependent(field.OriginalType, ctx) {
				t.Fatal("canonical Map lost its independent erased-storage proof")
			}
			plan, err := planGenericFamily(owner, ctx)
			if err != nil || plan == nil {
				t.Fatalf("independently supported Map field rejected: %v", err)
			}
			representation, present := plan.representations[parameter.Declaration]
			_, bound := plan.binders[parameter.Declaration]
			if !present || !bound || representation.erasure != "java.lang.Object" || representation.goName != "any" {
				t.Fatalf("unbounded T lost Object/any representation: %+v", representation)
			}
			physical := javaTypeStringToGoTypeExpr(field.OriginalType, inScopeTypeParameters(ctx), ctx)
			physical = genericFamilyPhysicalGoType(physical, owner.TypeParameters, ctx)
			if physical == nil || volatileFamilyRootExpr(t, physical) != "stdjava.JavaMap" {
				t.Fatalf("Map field must have non-indexed JavaMap interface storage: %T", physical)
			}
			descriptor, supported := javaTypeDescriptorExpr(field.OriginalType, ctx)
			if !supported || descriptor == nil || reflectCoreTypeID(t, descriptor) != "java.util.Map" {
				t.Fatal("Map lost its canonical logical descriptor")
			}
		})
	}
}

// Keep the original dependency source and prove both its canonical physical
// Class storage and its complete declaration-owned wildcard metadata.
func TestCanonicalClassPhysicalOriginalTypeTokenMetadataFrontier(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "canonical_class_physical129", "OriginalTypeToken", "TypeToken.java"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"com/google/gson/reflect/TypeToken.java": string(source)})
	owner := findQualifiedSourceClass("com.google.gson.reflect.TypeToken")
	ctx = classScopeCtx(owner, ctx)
	var raw *symbol.Definition
	for _, field := range owner.Fields {
		if field.OriginalName == "rawType" {
			raw = field
			break
		}
	}
	if raw == nil || raw.OriginalType != "Class<? super T>" {
		t.Fatalf("original field changed: %+v", raw)
	}
	if len(owner.TypeParameters) != 1 || raw.TypeParameterBindings["T"] != owner.TypeParameters[0].Declaration {
		t.Fatal("original field lost declaration-bound T")
	}
	var occurrence *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "field_declaration" {
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if child.Type() == "variable_declarator" && child.ChildByFieldName("name").Content(source) == "rawType" {
					occurrence = node.ChildByFieldName("type")
				}
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(owner.Class.DeclarationNode)
	if occurrence == nil || !canonicalClassArgumentIndependent(raw.OriginalType, ctx, occurrence) {
		t.Fatal("original canonical physical Class storage rejected")
	}
	tree, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: raw.OriginalType, TypeParameterBindings: raw.TypeParameterBindings}, ctx)
	if !supported {
		t.Fatal("original Class wildcard metadata rejected")
	}
	parameterized := reflectCoreComposite(t, tree)
	reflectCoreKind(t, parameterized, "ReflectParameterizedKind")
	if got := reflectCoreTypeID(t, reflectCoreKey(t, parameterized, "Raw")); got != "java.lang.Class" {
		t.Fatalf("original raw identity=%q", got)
	}
	arguments := reflectCoreSlice(t, reflectCoreKey(t, parameterized, "Arguments"))
	if len(arguments) != 1 {
		t.Fatal("original Class argument arity")
	}
	wildcard := reflectCoreComposite(t, arguments[0])
	reflectCoreKind(t, wildcard, "ReflectWildcardKind")
	upper, lower := reflectCoreSlice(t, reflectCoreKey(t, wildcard, "UpperBounds")), reflectCoreSlice(t, reflectCoreKey(t, wildcard, "LowerBounds"))
	if len(upper) != 1 || len(lower) != 1 {
		t.Fatal("original lower wildcard bounds")
	}
	reflectCoreClass(t, upper[0], "java.lang.Object")
	reflectMetadataVariable(t, lower[0], "com.google.gson.reflect.TypeToken", "T")
	if raw.OriginalType != "Class<? super T>" || raw.TypeParameterBindings["T"] != owner.TypeParameters[0].Declaration {
		t.Fatal("physical proof mutated logical metadata")
	}
}
