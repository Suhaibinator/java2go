# Known transpiler issues (e2e dedup reference)

Maintained by e2e-tester as the dedup source of truth for the differential fuzzer
(task #12) and anyone else triaging transpiler defects. Each entry is a distinct,
already-reported bug with a minimal Java repro and the observed failure. If a
fuzzer divergence reduces to one of these, it is a KNOWN bug — do not re-report;
cite the issue id here instead. If it does not match any entry, it is new.

Status legend: OPEN = still reproduces; FIXED = resolved, kept for history.
Last swept: HEAD 4ee1a98 (checkpoint 3 + Optional fixes).

---

## K1 — int/int32 typing (OPEN, ROADMAP §6, task #10 item 12) — HIGH IMPACT
Java `int` locals/expressions are emitted as Go `int` (or untyped constants), not
`int32`, so they clash with int32 fields/params/returns and don't wrap on overflow.
Blocks: lambdas, var_infer, numeric_edge (overflow line), collections/CollectionOps,
nested/AnonLocal (captured local), concurrency/ThreadJoin (entirely), concurrency/SyncCounter
(partly). 7 e2e fixtures — the single highest-leverage fix.
```java
public class K1 {
    static int f() { int count = 0; return count; } // count is Go int, return wants int32
    public static void main(String[] a) { System.out.println(f()); }
}
```
Also: `int max = 2147483647; max+1` prints 2147483648 (Go wide int) not -2147483648 (Java wrap).

## K2 — package-private type-name casing (OPEN, ROADMAP §6/#7, task #10 item 13) — HIGH IMPACT
Reference sites re-derive capitalization instead of using the resolved symbol's
generated name. A non-public class/interface/enum gets a lowercase struct but
references stay capitalized (and `New<lowercase>`), or vice-versa.
Blocks: inheritance (undefined Shape/Newrectangle/Newsquare), interfaces (undefined Greeter),
enums (method casing, see K3).
```java
class Animal { String s() { return "a"; } }
class Dog extends Animal {}
public class K2 { public static void main(String[] x){ System.out.println(new Dog().s()); } }
```

## K3 — method-name casing on generated types (OPEN, folded into K2 / item 13)
Call site emits the Java lowercase method name but the generated method is
capitalized. Seen on enum built-ins (`.ordinal()` vs `.Ordinal()`) AND on stdjava
runtime types: `stdjava.Optional` has `.Get()` but the call emits `.get()`. So this
isn't enum-specific — any generated/runtime type method call re-derives the lowercase
Java name instead of the generated capitalized name. Blocks enums and collections/Optionals.
```java
enum E { A, B; }
public class K3 { public static void main(String[] x){ System.out.println(E.A.ordinal()); } }
```

## K4 — char value prints as int code point (OPEN, ROADMAP §6)
A `char`-typed expression result (cast or arithmetic) prints its numeric code
point instead of the character. (The charAt intrinsic fix did NOT extend to cast
expressions.)
```java
public class K4 {
    public static void main(String[] a) {
        char c = 'A';
        System.out.println((char)(c + 1)); // prints 66, want B
    }
}
```

## K5 — long shift over-masked to 5 bits (OPEN, ROADMAP §6, task #10 item 11)
Shift amounts are masked to 5 bits for all shifts; long shifts must mask to 6.
```java
public class K5 { public static void main(String[] a){ System.out.println(1L << 32); } } // prints 1, want 4294967296
```

## K6 — user identifier collides with Go reserved/builtin name (FIXED for `init`, task #6)
A Java method named `init` now renames to `init0` at definition + all call sites
(classes-dev, resolve.go). NOT yet spot-checked for other collisions: `main`
(non-entry method), and builtins `len`/`cap`/`copy`/`new`/`delete`/`make`/`String`
used as identifiers — fuzzer-dev, worth probing these to see if the same rename
covers them or only `init` was special-cased.
```java
public class K6 {
    static int init(int x){ return x+1; }
    public static void main(String[] a){ System.out.println(init(4)); }
}
```

## K7 — method overload resolution collapses to first overload (OPEN, ROADMAP §6/§7)
Overloads get suffixed names (f0/f1...) but every call site picks the first; no
dispatch by argument type.
```java
public class K7 {
    static String f(int x){ return "int"; }
    static String f(String x){ return "str"; }
    public static void main(String[] a){ System.out.println(f(1)); System.out.println(f("a")); }
}
```

## K8 — var inferred from string concat loses String type (FIXED, task #13 item 1)
`var g = a + b` (String result) now records String type; var_simple passes.

## K9 — Optional empty()/of() inference (FIXED, task #3 / #13 item 1)
`Optional.empty()` inference, `Optional.of(literal)` element type, and `Optional.map`
all work now. Optionals' only remaining blocker is K3 method casing (.get/.Get).

## K10 — modern syntax: records, text blocks (OPEN, ROADMAP §5, task #6)
Records (`new Point(..)` -> undefined ConstructPoint) and text blocks (`"""..."""`
-> raw Go string with literal newlines) still unimplemented. Switch expressions and
instanceof pattern-binding now LOWER correctly (see K1 for switch_expr's remaining
int32 block, and K15 for the instanceof autoboxing gap).

## K15 — instanceof against a boxed primitive doesn't match (OPEN, ROADMAP §6 autoboxing)
`x instanceof String s` works (reference types match), but `obj instanceof Integer i`
where obj holds an autoboxed int (e.g. 21) does NOT match — the value is a Go
int/int32, not a boxed Integer, so the type assertion fails and it falls through.
```java
public class K15 {
    public static void main(String[] a) {
        Object o = 21;
        System.out.println(o instanceof Integer); // Java true; transpiled false
    }
}
```

## K11 — concurrency: Thread subclass + anonymous Runnable (FIXED, task #11)
All landed: `extends Thread`→goroutine, synchronized, and anonymous `Runnable`
(including inside a loop — struct now embeds `stdjava.Runnable`, `r.run()`→`r.Run()`).
Both concurrency fixtures (SyncCounter, ThreadJoin) are now blocked ONLY by K1 (int32)
and should pass the moment K1 lands. (The earlier `undefined: i` was the int32 literal
mismatch surfacing, not a capture-wiring bug — `i` was in scope.)

## K12 — compound `>>>=` emits undefined non-assigning call (OPEN, ROADMAP §6/§1) — found by fuzzer
The expression form `a >>> b` works (stdjava.UnsignedRightShift), but the compound
assignment `x >>>= n` emits `UnsignedRightShiftAssignment(x, n)` which is undefined
AND discards the result (never assigns x). Other compound-assign ops (`>>=`, `<<=`,
`&=` etc.) should be spot-checked too.
```java
public class K12 { public static void main(String[] a){ int x=-8; x >>>= 1; System.out.println(x); } } // want 2147483644
```

## K13 — unused local emitted without `_ = v` discard (OPEN, ROADMAP §1) — found by fuzzer
Java permits unused locals; Go rejects them ("declared and not used"). Method params
DO get a `_ = arg` discard, but local variable declarations do not.
```java
public class K13 { public static void main(String[] a){ long unused = 5L; System.out.println("ok"); } }
```
Generated: `unused := int64(5)` with no following `_ = unused` -> GO_COMPILE_ERROR.
