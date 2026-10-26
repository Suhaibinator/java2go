package transpiler

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestJDK21MemberDeclaredOrigin(t *testing.T) {
	for _, row := range []struct{ header, typ, want string }{
		{"import java.util.Map;", "Map.Entry<String,String>", "java.util.Map$Entry"},
		{"", "java.util.Map.Entry<String,Object>", "java.util.Map$Entry"},
		{"import java.util.Map.Entry;", "Entry<String,String>", "java.util.Map$Entry"},
		{"import java.util.Map.*;", "Entry<String,String>", "java.util.Map$Entry"},
		{"import static java.util.Map.Entry;", "Entry<String,String>", "java.util.Map$Entry"},
		{"import static java.util.Map.*;", "Entry<String,String>", "java.util.Map$Entry"},
		{"import java.util.AbstractMap;", "AbstractMap.Entry<String,String>", "java.util.Map$Entry"},
		{"import java.util.Map;", "Map.Entry<String,String>[]", "java.util.Map$Entry"},
		{"import java.lang.Thread;", "Thread.State", "java.lang.Thread$State"},
		{"import java.nio.channels.FileChannel;", "FileChannel.MapMode", "java.nio.channels.FileChannel$MapMode"},
	} {
		t.Run(row.header+row.typ, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, row.header+"class Probe{Object run("+row.typ+" value){return value;}}", "Probe")
			origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if origin.unresolved || origin.externalName != row.want || origin.nominalScope != nil || origin.parameter != nil {
				t.Fatalf("actual origin=%+v; want %s", origin, row.want)
			}
			if origin.externalName == "java.util.Map$Entry" {
				if len(origin.arguments) != 2 || origin.arguments[0].externalName != "java.lang.String" || origin.arguments[1].unresolved {
					t.Fatalf("lost actual own argument origins: %+v", origin)
				}
			}
		})
	}
}

func TestJDK21MemberNamespaceShields(t *testing.T) {
	for _, row := range []struct{ source, owner string }{
		{"import java.util.Map;class Probe<Map>{Object run(Map.Entry value){return value;}}", "Probe"},
		{"import java.util.Map;class Probe{<Map>Object run(Map.Entry value){return value;}}", "Probe"},
		{"import java.util.Map;class Outer<Map>{class Probe{Object run(Map.Entry value){return value;}}}", "Outer.Probe"},
		{"class Map{static class Entry{}}class Probe{Object run(Map.Entry value){return value;}}", "Probe"},
		{"class Map{}class Probe{Object run(Map.Entry value){return value;}}", "Probe"},
		{"import java.util.Map.Entry;class Probe{static class Entry{}Object run(Entry value){return value;}}", "Probe"},
		{"import foreign.Map;class Probe{Object run(Map.Entry value){return value;}}", "Probe"},
		{"import foreign.Entry;class Probe{Object run(Entry value){return value;}}", "Probe"},
		{"class java{}class Probe{Object run(java.util.Map.Entry value){return value;}}", "Probe"},
		{"import mystery.*;class Probe{Object run(java.util.Map.Entry value){return value;}}", "Probe"},
		{"import mystery.*;import java.util.Map.*;class Probe{Object run(Entry value){return value;}}", "Probe"},
		{"import static mystery.Owner.*;import java.util.Map.*;class Probe{Object run(Entry value){return value;}}", "Probe"},
		{"import java.util.Map.*;import java.security.KeyStore.*;class Probe{Object run(Entry value){return value;}}", "Probe"},
		{"class Probe{Object run(java.lang.String.CaseInsensitiveComparator value){return value;}}", "Probe"},
		{"class Probe{Object run(java.lang.StringTemplate.Processor value){return value;}}", "Probe"},
	} {
		t.Run(row.source, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, row.source, row.owner)
			origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if origin.externalName != "" {
				t.Fatalf("borrowed member identity: %+v", origin)
			}
		})
	}
	for _, missing := range []string{"file", "class", "stale", "detached"} {
		t.Run(missing, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, "class Probe{Object run(java.util.Map.Entry value){return value;}}", "Probe")
			switch missing {
			case "file":
				ctx.currentFile = nil
			case "class":
				ctx.currentClass = nil
			case "stale":
				ctx.currentFile = &symbol.FileScope{}
			case "detached":
				copy := *ctx.currentClass
				ctx.currentClass = &copy
			}
			if owner := declaredExternalNominalOwner("java.util.Map.Entry", nil, ctx); owner != "" {
				t.Fatalf("unowned declaration certified %s", owner)
			}
		})
	}
}

func TestJDK21MemberActualMetadata(t *testing.T) {
	for _, row := range []struct {
		owner, name, binary string
		state               jdk21NominalPresence
		eligible            bool
	}{
		{"java.util.Map", "Entry", "java.util.Map$Entry", jdk21NominalPresent, true},
		{"java.util.AbstractMap", "Entry", "java.util.Map$Entry", jdk21NominalPresent, true},
		{"java.util.Map", "Nonexistent", "", jdk21NominalAbsentCertified, false},
		{"java.util.Map.Entry", "Nonexistent", "", jdk21NominalUnknown, false},
		{"unknown.Owner", "Entry", "", jdk21NominalUnknown, false},
		{"java.lang.String", "CaseInsensitiveComparator", "java.lang.String$CaseInsensitiveComparator", jdk21NominalPresent, false},
	} {
		t.Run(row.owner+row.name, func(t *testing.T) {
			d, p := lookupJDK21Member(row.owner, row.name)
			if p != row.state || d.binaryName != row.binary || jdk21MemberEligible(d) != row.eligible {
				t.Fatalf("actual member %+v/%v", d, p)
			}
		})
	}
	data, err := os.ReadFile("testdata/jdk21nominal/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Members map[string]struct {
			Signature                                     string
			BinaryName                                    string
			DeclaringOwner                                string
			SimpleName                                    string
			Public, Static, Interface, Preview, Synthetic bool
			SourceChain                                   []string
			ClassEntrySHA, SourceEntrySHA                 string
		}
		Owners map[string]struct {
			CertificateID, Super string
			Interfaces           []string
			Members              map[string]string
		}
	}
	// Check the complete generated member set using actual JSON names.
	var document map[string]json.RawMessage
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(document["memberDeclarations"], &inventory.Members); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(document["memberOwners"], &inventory.Owners); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Members) != len(jdk21MemberDeclarations) || len(inventory.Owners) != len(jdk21MemberOwners) {
		t.Fatal("incomplete generated member/owner set")
	}
	for id, want := range inventory.Members {
		got, ok := jdk21MemberDeclarations[id]
		if !ok || got.binaryName != want.BinaryName || got.declaringOwner != want.DeclaringOwner || got.simpleName != want.SimpleName || got.public != want.Public || got.static != want.Static || got.isInterface != want.Interface || got.signature != want.Signature || !reflect.DeepEqual(got.sourceChain, want.SourceChain) || got.preview != want.Preview || got.synthetic != want.Synthetic || got.classSHA != want.ClassEntrySHA || got.sourceSHA != want.SourceEntrySHA {
			t.Fatalf("generated member mismatch %s", id)
		}
	}
	for id, want := range inventory.Owners {
		got, ok := jdk21MemberOwners[id]
		if !ok || got.certificate != want.CertificateID || got.superclass != want.Super || !reflect.DeepEqual(got.interfaces, want.Interfaces) || !reflect.DeepEqual(got.members, want.Members) {
			t.Fatalf("generated member owner mismatch %s", id)
		}
	}
	if _, p := lookupJDK21TopLevel("java.util.Map", "Entry"); p != jdk21NominalUnknown {
		t.Fatal("member metadata poisoned top-level absence")
	}
}

func TestJDK21MemberSourceInheritedAlias(t *testing.T) {
	for _, row := range []struct{ src, owner string }{
		{"abstract class Alias extends java.util.AbstractMap<String,String>{}class Probe{Object run(Alias.Entry<String,String> value){return value;}}", "Probe"},
		{"abstract class Probe extends java.util.AbstractMap<String,String>{Object run(Entry<String,String> value){return value;}}", "Probe"},
		{"abstract class Base extends java.util.AbstractMap<String,String>{}abstract class Alias extends Base{}class Probe{Object run(Alias.Entry<String,String> value){return value;}}", "Probe"},
		{"abstract class Alias extends java.util.AbstractMap<String,String> implements java.util.Map<String,String>{}class Probe{Object run(Alias.Entry<String,String> value){return value;}}", "Probe"},
	} {
		t.Run(row.src, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, row.src, row.owner)
			origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if origin.unresolved || origin.externalName != "java.util.Map$Entry" {
				t.Fatalf("actual inherited member origin=%+v", origin)
			}
		})
	}
}

func TestJDK21MemberInheritedAuthorityShields(t *testing.T) {
	for _, src := range []string{
		"abstract class Alias extends java.util.AbstractMap<String,String>{static class Entry{}}class Probe{Object run(Alias.Entry value){return value;}}",
		"abstract class Alias extends unknown.Parent{}class Probe{Object run(Alias.Entry value){return value;}}",
		"class Alias extends Other{}class Other extends Alias{}class Probe{Object run(Alias.Entry value){return value;}}",
		"interface Other{class Entry{}}abstract class Alias extends java.util.AbstractMap<String,String> implements Other{}class Probe{Object run(Alias.Entry value){return value;}}",
		"class Probe<Alias extends java.util.AbstractMap<String,String>>{Object run(Alias.Entry value){return value;}}",
	} {
		t.Run(src, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, src, "Probe")
			origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if origin.externalName != "" {
				t.Fatalf("lost actual inherited/source/binder authority: %+v", origin)
			}
		})
	}
}

func TestJDK21MemberDeclaredArgumentConversion(t *testing.T) {
	ctx := nativeChannelOriginContext(t, `import java.util.Map;
 class Cell<K,V> implements Map.Entry<K,V>{}class Child extends Cell<String,Integer>{}
 class Shaped{String getKey(){return "";}String getValue(){return "";}}
 class Probe{Object run(){return null;}}`, "Probe")
	for _, row := range []struct {
		actual, formal string
		applicable     bool
	}{
		{"Cell<String,Integer>", "Map.Entry<String,Integer>", true},
		{"Cell<String,Integer>", "Map.Entry<String,Object>", false},
		{"Child", "Map.Entry<String,Integer>", true},
		{"Child", "Map.Entry<String,String>", false},
		{"Child", "Map.Entry", true},
		{"Shaped", "Map.Entry<String,String>", false},
		{"Child[]", "Map.Entry<String,Integer>", false},
	} {
		t.Run(row.actual+row.formal, func(t *testing.T) {
			actual := declaredJavaTypeOrigin(symbol.JavaType{Original: row.actual}, ctx)
			formal := declaredJavaTypeOrigin(symbol.JavaType{Original: row.formal}, ctx)
			_, _, got, handled := declaredOriginConversion(actual, formal, ctx)
			if !handled || got != row.applicable {
				t.Fatalf("actual conversion handled%t applicable%t from %+v to %+v", handled, got, actual, formal)
			}
		})
	}
}

func TestJDK21MemberDonorArgumentOrigins(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"api/Donor.java":    `package api;import java.util.Map;class Donor{Map.Entry<String,String> call(Map.Entry<String,String> x){return x;}}`,
		"caller/Probe.java": `package caller;class Probe<Map,String>{}`,
	}, "caller/Probe.java")
	donor := findQualifiedSourceClass("api.Donor")
	if donor == nil {
		t.Fatal("missing actual donor")
	}
	def := donor.FindMethodByName("call", nil)
	if def == nil {
		t.Fatal("missing actual declaration")
	}
	origins := methodFormalOrigins(def, donor, nil, ctx)
	if len(origins) != 1 || origins[0].externalName != "java.util.Map$Entry" || len(origins[0].arguments) != 2 || origins[0].arguments[0].externalName != "java.lang.String" || origins[0].arguments[1].externalName != "java.lang.String" {
		t.Fatalf("borrowed caller/donor spelling: %+v", origins)
	}
}
