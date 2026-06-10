# Java → Go Conversion Roadmap

A checklist of missing or partial features needed to support full Java-to-Go conversion,
roughly in suggested priority order. See file references for where each gap lives today.

> **Status (2026-06-10):** Work in progress by team `java2go-roadmap`.
> §1 robustness-dev · §2 stdlib-dev · §3 exceptions-dev · §4–5 classes-dev · §6–7 queued ·
> E2E test suite (testfiles/e2e/) owned by e2e-tester. Checkboxes are ticked as work is
> reviewed and merged, not when first submitted.

## 1. Robustness / graceful degradation

- [ ] Replace `panic("Unhandled expression: ...")` in `ParseExpr` with a diagnostic + recoverable error (`transpiler/expression.go:797`)
- [ ] Replace `panic` in `ParseStmt` with error recovery (`transpiler/statement.go:20`)
- [ ] Replace `panic` in `ParseDecl` with error recovery (`transpiler/declaration.go:1475`)
- [ ] Replace `panic` in `ParseNode` catch-all (`transpiler/tree_sitter.go:339`)
- [ ] Emit `// UNSUPPORTED: <construct>` stubs (or collect diagnostics) so one unknown construct doesn't abort the whole conversion
- [ ] Add a `-strict` flag to opt back into fail-fast behavior

## 2. Standard library mapping

- [ ] Build a systematic intrinsics table (Java method → Go equivalent) instead of one-off cases in `transpiler/expression.go:276`
- [ ] `java.lang.String` methods: `substring`, `split`, `equals`, `toUpperCase`, `toLowerCase`, `charAt`, `indexOf`, `trim`, `replace`, `format`, ...
- [ ] `StringBuilder` / `StringBuffer` → `strings.Builder`
- [ ] Collections: `ArrayList`, `LinkedList`, `HashMap`, `TreeMap`, `HashSet`, `TreeSet` (slices/maps or `stdjava` shim types)
- [ ] `java.util.Collections` utilities (`sort`, `reverse`, `max`, `min`, ...)
- [ ] Iterators / `Iterable` protocol mapping
- [ ] Wire `java.util.Optional` to the existing (currently unused) `stdjava/optional.go` stub
- [ ] `java.util.stream.*` (Streams API) — map to loops or a `stdjava` stream shim
- [ ] `java.io.*` basics (`File`, readers/writers, `Scanner`) → `os` / `bufio` / `io`
- [ ] `java.lang.Math`, boxed-type statics (`Integer.parseInt`, `Long.MAX_VALUE`, ...)
- [ ] Expand the `stdjava` runtime package to back anything with no direct Go analogue

## 3. Exception semantics

- [ ] Model the `Throwable → Exception → RuntimeException` hierarchy so catch-by-supertype dispatches correctly (`transpiler/tree_sitter.go:560`)
- [ ] Preserve exception messages and types in the panic value (currently lost as `any`)
- [ ] Map common exception types (`NullPointerException`, `IllegalArgumentException`, ...) to `stdjava` types
- [ ] Handle `throws` clauses (currently ignored) — at minimum document, ideally translate to Go error returns as an option
- [ ] Support `e.getMessage()` / `e.printStackTrace()` on caught exceptions

## 4. Inner / nested / anonymous classes

- [ ] Inner (non-static) classes — TODO at `transpiler/expression.go:449` (`parent.new Nested()`)
- [ ] Static nested classes
- [ ] Anonymous classes implementing a SAM interface (reuse the existing functional-interface adapter machinery in `transpiler/declaration.go:853`)
- [ ] Anonymous classes with multiple methods / extending a class
- [ ] Local classes declared inside methods

## 5. Modern Java syntax (Java 10–17+)

- [ ] `var` local variable type inference
- [ ] Switch expressions with `->` arms and `yield` (Java 12+)
- [ ] `instanceof` pattern matching: `if (x instanceof String s)` (Java 16+)
- [ ] Records (Java 14+)
- [ ] Sealed classes/interfaces (Java 15+)
- [ ] Text blocks `"""..."""` (Java 13+)

## 6. Semantic fidelity in existing features

- [ ] Implicit numeric widening at call sites (`int` arg → `long` param)
- [ ] Autoboxing/unboxing (`int` ↔ `Integer`)
- [ ] Implicit array-to-varargs conversion at call sites (declarations already work, `transpiler/tree_sitter.go:312`)
- [ ] `? super T` wildcard currently approximated as `any` — preserve the bound where possible (`README.md:52`)
- [ ] Static initializer ordering guarantees matching Java class-loading order
- [ ] Verify integer overflow / division / shift semantics match Java
- [ ] Covariant return types in overridden methods

## 7. Concurrency

- [ ] `synchronized` blocks/methods → mutexes (currently silently stripped, `transpiler/tree_sitter.go:261`) — at minimum warn instead of dropping
- [ ] `Thread` / `Runnable` → goroutines
- [ ] `volatile` → atomics
- [ ] `java.util.concurrent.*` basics (`ExecutorService`, `ConcurrentHashMap`, `AtomicInteger`, ...)
- [ ] `wait`/`notify` → channels or condition variables

## 8. Out of scope / decide explicitly

- [ ] Reflection (`getClass()`, `Class.forName()`, `java.lang.reflect.*`) — decide whether to support a subset or document as unsupported
- [ ] Module system (`module-info.java`, Java 9+)
- [ ] Class loading semantics / custom class loaders
