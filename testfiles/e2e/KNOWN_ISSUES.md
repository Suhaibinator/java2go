# Known transpiler issues (e2e dedup reference)

Maintained by e2e-tester as the dedup source of truth for the differential fuzzer
(task #12) and anyone else triaging transpiler defects. Each entry is a distinct,
already-reported bug with a minimal Java repro and the observed failure. If a
fuzzer divergence reduces to one of these, it is a KNOWN bug — do not re-report;
cite the issue id here instead. If it does not match any entry, it is new.

Status legend: OPEN = still reproduces; FIXED = resolved, kept for history.
Last swept: 2026-07-20 on `newBranch` (strict e2e and application-parity sweep).

---

## K1 — int/int32 typing (FIXED)
Declared and inferred Java `int` locals are pinned to `int32`; numeric promotion,
parameters, fields, returns, and array boxing preserve the same width. The strict
numeric-edge fixture verifies overflow wrapping as well as mixed arithmetic.
```java
public class K1 {
    static int f() { int count = 0; return count; } // count is Go int, return wants int32
    public static void main(String[] a) { System.out.println(f()); }
}
```
Also: `int max = 2147483647; max+1` prints 2147483648 (Go wide int) not -2147483648 (Java wrap).

## K2 — package-private type-name casing (FIXED)
Reference, constructor, superclass, interface, and array sites now use resolved
generated names. The inheritance and interface fixtures are enforced strict parity.
```java
class Animal { String s() { return "a"; } }
class Dog extends Animal {}
public class K2 { public static void main(String[] x){ System.out.println(new Dog().s()); } }
```

## K3 — method-name casing on generated types (FIXED)
Call sites resolve the generated method symbol, including enum/runtime methods and
cross-package generic receivers. Enums and Optionals are enforced strict parity.
```java
enum E { A, B; }
public class K3 { public static void main(String[] x){ System.out.println(E.A.ordinal()); } }
```

## K4 — char value prints as int code point (FIXED)
`char`-typed expressions are converted to text for print/concatenation, while
ordinary arithmetic still promotes them numerically. Compound narrowing wraps
through unsigned 16-bit Java char range.
```java
public class K4 {
    public static void main(String[] a) {
        char c = 'A';
        System.out.println((char)(c + 1)); // prints 66, want B
    }
}
```

## K5 — long shift over-masked to 5 bits (FIXED)
Shift distance and unsigned-shift width now distinguish 32-bit int from 64-bit long.
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

## K7 — method overload resolution collapses to first overload (FIXED)
Call sites now select the distinct generated overload name by arity and inferred
Java argument type. Exact primitive/reference matches win first, followed by
legal Java numeric widening; typed locals retain their declared static type for
selection. `overloading/Overloading` is enforced as byte-identical Java/Go parity.
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
all work now, including resolved generated method casing.

## K10 — modern syntax: records, text blocks (FIXED)
Records, text blocks, switch expressions, and instanceof pattern binding all have
strict byte-parity fixtures.

## K15 — instanceof against a boxed primitive doesn't match (FIXED)
Java-width primitive boxing is preserved when values enter Object locals/arrays,
so later Integer/Long/Float/Double pattern assertions observe the expected type.
```java
public class K15 {
    public static void main(String[] a) {
        Object o = 21;
		System.out.println(o instanceof Integer); // true in both runtimes
    }
}
```

## K11 — concurrency: Thread subclass + anonymous Runnable (FIXED, task #11)
All landed: `extends Thread`→goroutine, synchronized, and anonymous `Runnable`
(including inside a loop — struct now embeds `stdjava.Runnable`, `r.run()`→`r.Run()`).
Both concurrency fixtures (SyncCounter, ThreadJoin) pass strict parity.

## K12 — compound `>>>=` emits undefined non-assigning call (FIXED)
Value-producing and statement-form assignments use single-evaluation lowering,
Java shift masks, and width-preserving int/long unsigned shift helpers.
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

## K14 — readLine loop target cannot represent null at EOF (OPEN, String nullability)
The canonical `while ((line = reader.readLine()) != null)` loop correctly
distinguishes empty lines from EOF, but Java assigns null to `line` on the final
condition check. Generated Go represents String as a non-nullable `string`, so the
bridge leaves the previous value after the loop. Code that observes `line` after
EOF therefore still differs.
```java
String line = "before";
while ((line = reader.readLine()) != null) {}
System.out.println(line == null); // Java true; generated model cannot express it
```

## K16 — overloaded constructor conversion is exact-only (OPEN, overload resolution)
Method overloads use Java widening and most-specific reference selection, but
constructor selection still primarily matches exact parameter types/arity. A
constructor call that needs numeric widening or a reference upcast can therefore
choose incorrectly or fail to resolve. This is retained as a TDD target.

## K17 — user `toString()` ignored by the plain StringValueOf bridge (OPEN, string conversion)
`stdjava.StringValueOf` (`stdjava/string_conversion.go`) falls through to
`fmt.Sprint` for any non-float value, so a generated class's `toString()` is never
consulted. Printing an object, or a collection holding one, yields Go's struct
rendering instead of the Java text. The execution-aware variant
`StringValueOfExecution` does bridge it (via the `executionStringer` interface and
the reflective collision-safe path), but plain print sites and every collection's
`String()` method call the non-execution form.
```java
public class K17 {
    static class P { int v; P(int v){ this.v = v; } public String toString(){ return "P" + v; } }
    public static void main(String[] a) {
        System.out.println(new P(3));                       // Java P3, generated &{3}
        java.util.List<P> l = new java.util.ArrayList<P>();
        l.add(new P(3));
        System.out.println(l);                              // Java [P3], generated [&{3}]
    }
}
```
Fixing it needs a decision about how collection `String()` methods, which hold no
execution token, reach an execution-aware generated stringer.

## K18 — three-argument `Stream.reduce` is unsupported (FIXED)
`<U> U reduce(U identity, BiFunction<U,T,U> accumulator, BinaryOperator<U> combiner)`
gives its two lambdas different parameter shapes — `(U, T)` and `(U, U)` — and a
result type unrelated to the stream's element type. The element-typed lambda
machinery in `transpiler/intrinsics.go` assigns one element type to every
parameter of every lambda argument, so it cannot express this.

Fixed by the per-argument lambda typing added for Collectors
(`registerLambdaArgumentTyper` in `transpiler/intrinsics.go`), which gives each
lambda argument of a call its own parameter and result types instead of one
shared element type. `U` is read from the identity argument.
```java
List<Integer> nums = List.of(1, 2, 3);
String s = nums.stream().reduce("", (acc, x) -> acc + x, (a, b) -> a + b); // Java "123"
```
Java offers the overload to merge partial results across parallel splits, and
this runtime evaluates sequentially, so the combiner is accepted and never
invoked.
