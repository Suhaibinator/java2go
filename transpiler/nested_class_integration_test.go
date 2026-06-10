package transpiler

import (
	"strings"
	"testing"
)

// --- Static nested classes ---

func TestStaticNestedClass_EmittedAsTopLevelStruct(t *testing.T) {
	src := `
package nested;
public class Outer {
    public static class Inner {
        public int value;
        public int getValue() { return value; }
    }
    public Inner make() { return new Inner(); }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type OuterInner struct") {
		t.Fatalf("expected static nested class to be emitted as a collision-safe top-level struct OuterInner, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewOuterInner() *OuterInner") {
		t.Fatalf("expected a synthesized default constructor NewOuterInner, got:\n%s", out)
	}
	if !strings.Contains(flat, "return NewOuterInner()") {
		t.Fatalf("expected `new Inner()` to resolve to the renamed nested-class constructor, got:\n%s", out)
	}
}

func TestStaticNestedClass_WithExplicitConstructor(t *testing.T) {
	src := `
package nested;
public class Container {
    public static class Item {
        public int id;
        public Item(int id) { this.id = id; }
    }
    public Item create(int n) { return new Item(n); }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func NewContainerItem(id int32) *ContainerItem") {
		t.Fatalf("expected explicit nested-class constructor to be renamed NewContainerItem, got:\n%s", out)
	}
	if !strings.Contains(flat, "return NewContainerItem(n)") {
		t.Fatalf("expected `new Item(n)` to call the renamed nested-class constructor, got:\n%s", out)
	}
}

func TestStaticNestedClass_RuntimeBehavior(t *testing.T) {
	src := `
package nested;
public class Registry {
    public static class Entry {
        public int key;
        public int weight = 10;
        public Entry(int key) { this.key = key; }
        public int score() { return key * weight; }
    }
    public static int run() {
        Entry e = new Entry(4);
        return e.score();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestStaticNestedRuntime(t *testing.T) {
	got := Run()
	if got != 40 {
		t.Fatalf("Run() = %d, want 40 (key=4 * weight=10)", got)
	}
}
`)
}

// A constructor-less concrete class instantiated with `new` should still get a
// usable synthesized default constructor, with instance-field initializers run.
func TestDefaultConstructor_RunsFieldInitializers(t *testing.T) {
	src := `
package nested;
public class Widget {
    public int width = 7;
    public int getWidth() { return width; }
    public static int run() {
        Widget w = new Widget();
        return w.getWidth();
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "func NewWidget() *Widget") {
		t.Fatalf("expected synthesized default constructor for constructor-less class, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestDefaultCtorFieldInit(t *testing.T) {
	got := Run()
	if got != 7 {
		t.Fatalf("Run() = %d, want 7 (field initializer should run in default ctor)", got)
	}
}
`)
}

// --- Inner (non-static nested) classes ---

func TestInnerClass_EnclosingInstanceFieldAndAccess(t *testing.T) {
	src := `
package nested;
public class Outer {
    private int base = 100;
    public class Inner {
        public int delta;
        public int total() { return base + delta; }
    }
    public Inner make() { return new Inner(); }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type OuterInner struct { outer *Outer") {
		t.Fatalf("expected inner class to carry a synthesized enclosing-instance field, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewOuterInner(outer *Outer) *OuterInner") {
		t.Fatalf("expected inner-class constructor to capture the enclosing instance, got:\n%s", out)
	}
	if !strings.Contains(flat, "or.outer = outer") {
		t.Fatalf("expected inner-class constructor to store the enclosing instance, got:\n%s", out)
	}
	if !strings.Contains(flat, "or.outer.base + or.Delta") {
		t.Fatalf("expected unqualified outer-field access to route through the enclosing instance, got:\n%s", out)
	}
	if !strings.Contains(flat, "return NewOuterInner(or)") {
		t.Fatalf("expected implicit `new Inner()` to pass the current receiver as enclosing instance, got:\n%s", out)
	}
}

func TestInnerClass_ExplicitOuterNewQualifier(t *testing.T) {
	src := `
package nested;
public class Outer {
    public class Inner {
        public int v;
    }
    public static Inner build(Outer o) { return o.new Inner(); }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "return NewOuterInner(o)") {
		t.Fatalf("expected `o.new Inner()` to pass the qualifier as enclosing instance, got:\n%s", out)
	}
}

func TestInnerClass_RuntimeBehavior(t *testing.T) {
	src := `
package nested;
public class Outer {
    private int base = 100;
    public int bump() { return base + 1; }
    public class Inner {
        public int delta = 5;
        public int total() { return base + delta + bump(); }
    }
    public static int run() {
        Outer o = new Outer();
        Outer.Inner in = o.new Inner();
        return in.total();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInnerRuntime(t *testing.T) {
	got := Run()
	// base(100) + delta(5) + bump()(base+1 = 101) = 206
	if got != 206 {
		t.Fatalf("Run() = %d, want 206", got)
	}
}
`)
}

func TestInnerClass_CapturesMutatedOuterState_Runtime(t *testing.T) {
	src := `
package nested;
public class Counter {
    private int count = 0;
    public void inc() { count = count + 1; }
    public class Reader {
        public int read() { return count; }
    }
    public static int run() {
        Counter c = new Counter();
        Reader r = c.new Reader();
        c.inc();
        c.inc();
        c.inc();
        return r.read();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInnerSharedState(t *testing.T) {
	got := Run()
	// The inner Reader observes the enclosing Counter's mutations through the
	// shared enclosing-instance pointer.
	if got != 3 {
		t.Fatalf("Run() = %d, want 3 (inner class should observe outer mutations)", got)
	}
}
`)
}

// --- Anonymous classes implementing a single-abstract-method interface ---

func TestAnonymousClass_SAMInterface_LowersToAdapter(t *testing.T) {
	src := `
package nested;
public interface Greeter { String greet(String name); }
public class App {
    public static Greeter make() {
        return new Greeter() {
            public String greet(String name) { return "hi " + name; }
        };
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "return NewGreeterFuncAdapter(func(name string) string") {
		t.Fatalf("expected anonymous SAM class to lower to the functional-interface adapter with a closure, got:\n%s", out)
	}
}

func TestAnonymousClass_SAMInterface_RuntimeBehavior(t *testing.T) {
	src := `
package nested;
public interface IntOp { int apply(int x); }
public class App {
    public static int withBase(int base) {
        IntOp op = new IntOp() {
            public int apply(int x) { return x + base; }
        };
        return op.apply(5);
    }
    public static int run() {
        return withBase(10);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestAnonSAMRuntime(t *testing.T) {
	got := Run()
	// captured base(10) + arg(5) = 15
	if got != 15 {
		t.Fatalf("Run() = %d, want 15 (anonymous class should capture the enclosing parameter)", got)
	}
}
`)
}

// --- Anonymous classes with multiple methods / extending a class ---

func TestAnonymousClass_MultiMethod_SynthesizedStruct(t *testing.T) {
	src := `
package nested;
public interface Shape { int area(); int perimeter(); }
public class App {
    public static Shape make(int w, int h) {
        return new Shape() {
            public int area() { return w * h; }
            public int perimeter() { return 2 * (w + h); }
        };
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type AppAnon1 struct { Shape") {
		t.Fatalf("expected a synthesized struct embedding the interface for a multi-method anonymous class, got:\n%s", out)
	}
	if !strings.Contains(flat, "return &AppAnon1{w: w, h: h}") {
		t.Fatalf("expected the creation site to build the synthesized struct capturing locals, got:\n%s", out)
	}
}

func TestAnonymousClass_MultiMethod_RuntimeBehavior(t *testing.T) {
	src := `
package nested;
public interface Shape { int area(); int perimeter(); }
public class App {
    public static Shape make(int w, int h) {
        return new Shape() {
            public int area() { return w * h; }
            public int perimeter() { return 2 * (w + h); }
        };
    }
    public static int run() {
        Shape s = make(3, 4);
        return s.area() + s.perimeter();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestAnonMultiRuntime(t *testing.T) {
	got := Run()
	// area(3*4=12) + perimeter(2*(3+4)=14) = 26
	if got != 26 {
		t.Fatalf("Run() = %d, want 26", got)
	}
}
`)
}

// --- Local classes (declared inside a method) ---

func TestLocalClass_HoistedToFileScope(t *testing.T) {
	src := `
package nested;
public class App {
    public static int run(int factor) {
        class Multiplier {
            int times(int x) { return x * factor; }
        }
        Multiplier m = new Multiplier();
        return m.times(5);
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type AppLocalMultiplier1 struct { factor int32 }") {
		t.Fatalf("expected local class to be hoisted to a file-scope struct capturing the enclosing local, got:\n%s", out)
	}
	if !strings.Contains(flat, "m := &AppLocalMultiplier1{factor: factor}") {
		t.Fatalf("expected `new Multiplier()` to build the hoisted struct with captures, got:\n%s", out)
	}
}

func TestLocalClass_RuntimeBehavior(t *testing.T) {
	src := `
package nested;
public class App {
    public static int compute(int factor) {
        class Multiplier {
            int times(int x) { return x * factor; }
        }
        Multiplier m = new Multiplier();
        return m.times(5);
    }
    public static int run() {
        return compute(3);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestLocalClassRuntime(t *testing.T) {
	got := Run()
	// times(5) with captured factor(3) = 15
	if got != 15 {
		t.Fatalf("Run() = %d, want 15 (local class should capture the enclosing local)", got)
	}
}
`)
}

// A top-level class whose name collides with a nested class's concatenated name
// (Outer + Inner == OuterInner) must be disambiguated so the generated Go has no
// duplicate type/constructor declarations.
func TestNestedClass_NameCollisionWithTopLevel(t *testing.T) {
	src := `
package nested;
public class Outer {
    public static class Inner {
        public int a;
    }
}
public class OuterInner {
    public int b;
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	// The nested class claims OuterInner; the top-level one is suffixed.
	if !strings.Contains(flat, "type OuterInner struct { A int32 }") {
		t.Fatalf("expected nested Outer.Inner to keep the OuterInner name, got:\n%s", out)
	}
	if !strings.Contains(flat, "type OuterInner2 struct { B int32 }") {
		t.Fatalf("expected colliding top-level OuterInner to be disambiguated, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewOuterInner2() *OuterInner2") {
		t.Fatalf("expected the disambiguated class's constructor to be retargeted, got:\n%s", out)
	}
	if strings.Count(flat, "type OuterInner struct") != 1 {
		t.Fatalf("expected exactly one `type OuterInner struct` declaration, got:\n%s", out)
	}
}
