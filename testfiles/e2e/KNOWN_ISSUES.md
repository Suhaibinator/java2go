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
capitalized. Seen on enum built-ins: `.ordinal()` vs generated `.Ordinal()`.
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

## K6 — user identifier collides with Go reserved/builtin name (OPEN, ROADMAP §4 naming)
A Java method named `init` becomes Go `func init(...)`, which Go reserves for
no-arg/no-return package initializers -> invalid. Likely also `main` (non-entry),
and builtins `len`/`cap`/`copy`/`new`/`delete` used as identifiers.
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

## K8 — var inferred from string concat loses String type (OPEN, ROADMAP §2/§6)
`var g = a + b` where the result is String produces a Sprintf whose type isn't
recorded as String, so String intrinsics on `g` aren't dispatched.
```java
public class K8 { public static void main(String[] a){ var g = "ab"+"cd"; System.out.println(g.length()); } }
```

## K9 — Optional gaps (OPEN, ROADMAP §2/§3)
`Optional.empty()` can't infer T (no type context); `Optional.of(literal)` infers
element type `any` not the literal's type. (`Optional.map` was recently added.)
```java
import java.util.Optional;
public class K9 {
    static Optional<String> e(){ return Optional.empty(); }
    public static void main(String[] a){ System.out.println(e().orElse("x")); }
}
```

## K10 — modern syntax unimplemented (OPEN, ROADMAP §5, task #6)
Switch expressions (`switch(x){case ... -> ...}` emit panic() in value position),
instanceof patterns (`x instanceof String s`), records (ConstructX undefined),
text blocks (`"""..."""` -> raw string with literal newlines). All in progress.

## K11 — concurrency: anonymous Runnable inside a loop (OPEN, ROADMAP §7, task #11)
PARTIALLY FIXED in checkpoint 4: `extends Thread`→goroutine and synchronized work
now (ThreadJoin compiles modulo K1). Remaining: an anonymous `Runnable` created
inside a loop emits `undefined: Runnable` and `undefined: i` (the loop var captured
into the anon body isn't wired). Seen in concurrency/SyncCounter.
```java
public class K11 {
    public static void main(String[] a) {
        for (int i = 0; i < 2; i++) {
            Runnable r = new Runnable() { public void run() { System.out.println(i); } };
            r.run();
        }
    }
}
```

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
