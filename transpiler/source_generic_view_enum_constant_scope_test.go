package transpiler

import "testing"

// Independent JDK21 compiled these exact two Java sources and ran seeds
// 17/41/97 twice: Choice.A owns its Cell namespace, while Main uses p.Cell.
func TestSourceGenericViewDemandEnumConstantAnonymousScope(t *testing.T) {
    ctx := sourceGenericViewDemandTestContext(t, map[string]string{
        "p/Cell.java": `package p;
public final class Cell<T> {
    public T value;
    public Cell(T value) { this.value = value; }
}
`,
        "app/Main.java": `package app;
import p.Cell;
enum Choice {
    A {
        static final class Cell<T> {
            String marker() { return "constant"; }
        }
        Cell<?> slot = new Cell<String>();
        int calls;
        @Override String marker() {
            calls++;
            return slot.marker() + calls;
        }
    };
    abstract String marker();
}
public final class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Cell<String> external = new Cell<>("seed" + seed);
        System.out.println(Choice.A.marker() + ":" + Choice.A.marker() + ":"
            + external.value + ":" + (Choice.A == Choice.valueOf("A")));
    }
}
`,
    })
    external := findQualifiedSourceClass("p.Cell")
    if external == nil { t.Fatal("independent fixture lost external source declaration") }
    for _, seed := range sourceGenericViewDemandSeeds(ctx) {
        if seed == external {
            t.Error("enum constant anonymous-body Cell<?> selected imported p.Cell")
        }
    }
    if canonicalGenericFamily(external, ctx) != nil {
        t.Error("enum constant anonymous-body shadow spuriously migrated p.Cell")
    }
}
