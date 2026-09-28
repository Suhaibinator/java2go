package transpiler

import "testing"

func TestGenericFamilyPlanInventoriesConnectedDeclarations(t *testing.T) {
	helper := setupParseHelper(t, `
interface Slot<T>{T read();void write(T value);}
abstract class Adapter<T> implements Slot<T>{
 public abstract T read();public abstract void write(T value);
 Adapter<T> nullSafe(){return new NullSafe();}
 private final class NullSafe extends Adapter<T>{public T read(){return Adapter.this.read();}public void write(T value){Adapter.this.write(value);}}
}
abstract class Middle<X> extends Adapter<X>{X twice(){read();return read();}}
class Memory<T> extends Middle<T>{T value;public T read(){return value;}public void write(T value){this.value=value;}}
class Specific extends Middle<String>{public String read(){return "";}public void write(String value){}}
class Use{Adapter<String> make(){return new Adapter<String>(){String state;public String read(){return state;}public void write(String value){state=value;}};}}
`)
	seed := helper.File.Symbols.FindClassScope("Adapter")
	plan, err := planGenericFamily(seed, helper.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Slot", "Adapter", "Middle", "Memory", "Specific"} {
		if _, present := plan.members[helper.File.Symbols.FindClassScope(name)]; !present {
			t.Errorf("missing connected declaration %s", name)
		}
	}
	if len(plan.members) != 6 {
		t.Fatalf("named inventory = %d, want 6 including inner class", len(plan.members))
	}
	if len(plan.anonymous) != 1 {
		t.Fatalf("anonymous inventory = %d, want 1", len(plan.anonymous))
	}
	for key, anonymous := range plan.anonymous {
		if key.file != helper.File.Symbols || anonymous.parent != seed || anonymous.javaType != "Adapter<String>" {
			t.Fatalf("anonymous source identity lost: %#v", anonymous)
		}
	}
}

func TestGenericFamilyPlanRejectsPartialStorageMigration(t *testing.T) {
	for _, test := range []struct{ name, body, extra string }{
		{name: "runtime field", body: "java.util.List<T> values;"},
		{name: "runtime method", body: "java.util.Map<String,T> values(){return null;}"},
		{name: "runtime local", body: "T read(){java.util.List<T> values=null;return null;}"},
		{name: "array slot", body: "T[] values;"},
		{name: "constructor typed local", body: "T value; Adapter(T value){this.value=value;T copy=this.value;}"},
		{name: "invariant specialized descendant", extra: "abstract class Bad extends Adapter<java.util.List<String>>{}"},
		{name: "invariant anonymous descendant", extra: "class Use{Object make(){return new Adapter<java.util.List<String>>(){};}}"},
		{name: "local descendant", extra: "class Use{void make(){class Local extends Adapter<String>{}}}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, `abstract class Adapter<T>{`+test.body+`}`+test.extra)
			seed := helper.File.Symbols.FindClassScope("Adapter")
			if plan, err := planGenericFamily(seed, helper.Ctx); err == nil || plan != nil {
				t.Fatalf("incomplete family admitted: plan=%#v err=%v", plan, err)
			}
		})
	}
}

func TestGenericFamilyPlanUsesDeclarationIdentity(t *testing.T) {
	helper := setupParseHelper(t, `abstract class Adapter<T>{T value;<T> java.util.List<T> shadow(java.util.List<T> value){return value;}}`)
	seed := helper.File.Symbols.FindClassScope("Adapter")
	if _, err := planGenericFamily(seed, helper.Ctx); err != nil {
		t.Fatalf("method binder confused with owning class binder: %v", err)
	}
}

func TestGenericFamilyPlanClosesSourceStorageDependencies(t *testing.T) {
	for _, unsupported := range []bool{false, true} {
		extra := ""
		if unsupported {
			extra = "java.util.List<V> forbidden;"
		}
		helper := setupParseHelper(t, `class Adapter<T>{Cell<T> cell;} class Cell<U>{Link<U> next;} class Link<V>{Cell<V> previous;`+extra+`}`)
		seed := helper.File.Symbols.FindClassScope("Adapter")
		plan, err := planGenericFamily(seed, helper.Ctx)
		if unsupported {
			if err == nil || plan != nil {
				t.Fatal("cyclic storage dependency admitted an invariant runtime List")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"Adapter", "Cell", "Link"} {
			if _, ok := plan.members[helper.File.Symbols.FindClassScope(name)]; !ok {
				t.Errorf("missing transitive storage dependency %s", name)
			}
		}
	}
}
