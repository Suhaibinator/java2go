# Java → Go Conversion Roadmap

A checklist of missing or partial features needed to support full Java-to-Go conversion,
roughly in suggested priority order. See file references for where each gap lives today.

> **Status (2026-06-10):** Work in progress by team `java2go-roadmap`.
> §1 robustness-dev · §2 stdlib-dev · §3 exceptions-dev · §4–5 classes-dev · §6–7 queued ·
> E2E test suite (testfiles/e2e/) owned by e2e-tester. Checkboxes are ticked as work is
> reviewed and merged, not when first submitted.

## 1. Robustness / graceful degradation

- [x] Replace `panic("Unhandled expression: ...")` in `ParseExpr` with a diagnostic + recoverable error
- [x] Replace `panic` in `ParseStmt` with error recovery
- [x] Replace `panic` in `ParseDecl` with error recovery
- [x] Replace `panic` in `ParseNode` catch-all
- [x] Emit `// UNSUPPORTED: <construct>` stubs (collected as thread-safe diagnostics; see `transpiler/diagnostics.go`)
- [x] Add a `-strict` flag to opt back into fail-fast behavior (CLI + library API via `java2go.Diagnostics()`)

## 2. Standard library mapping

- [x] Build a systematic intrinsics table (Java method → Go equivalent) — `transpiler/intrinsics.go` (instance/static/static-field/constructor registries)
- [x] `java.lang.String` methods: rune-safe `substring`, `split` (literal separators; regex patterns TODO), `equals`, `toUpperCase`, `toLowerCase`, `charAt`, `indexOf`, `trim`, `replace`, `format`, ... (`stdjava/strings.go`)
- [x] `StringBuilder` / `StringBuffer` → `stdjava` wrapper (`stdjava/stringbuilder.go`)
- [ ] Collections: `ArrayList`, `LinkedList`, `HashMap`, `TreeMap`, `HashSet`, `TreeSet` (slices/maps or `stdjava` shim types)
- [ ] `java.util.Collections` utilities (`sort`, `reverse`, `max`, `min`, ...)
- [ ] Iterators / `Iterable` protocol mapping
- [ ] Wire `java.util.Optional` to the existing (currently unused) `stdjava/optional.go` stub
- [ ] `java.util.stream.*` (Streams API) — map to loops or a `stdjava` stream shim
- [ ] `java.io.*` basics (`File`, readers/writers, `Scanner`) → `os` / `bufio` / `io`
- [x] `java.lang.Math`, boxed-type statics (`Integer.parseInt`, `Long.MAX_VALUE`, ...) — type-preserving generics in `stdjava/math.go`, `stdjava/convert.go`
- [x] Expand the `stdjava` runtime package to back anything with no direct Go analogue (ongoing as features land)

## 3. Exception semantics

- [x] Model the `Throwable → Exception → RuntimeException` hierarchy so catch-by-supertype dispatches correctly (name-based registry; `stdjava.CaughtAs`, user types register via generated `init()`)
- [x] Preserve exception messages and types in the panic value (`stdjava/exceptions.go` typed values; runtime panics normalized via `stdjava.NormalizePanic`)
- [x] Map common exception types (`NullPointerException`, `IllegalArgumentException`, ...) to `stdjava` types
- [x] Handle `throws` clauses — preserved as `// throws ...` doc comments (error-return translation deliberately out of scope)
- [x] Support `e.getMessage()` / `e.printStackTrace()` on caught exceptions

## 4. Inner / nested / anonymous classes

- [x] Inner (non-static) classes — synthesized enclosing-instance field, `outer.new Inner()`, outer-member access through the enclosing chain
- [x] Static nested classes — emitted as `OuterInner` top-level structs with retargeted constructors
- [ ] Anonymous classes implementing a SAM interface (reuse the existing functional-interface adapter machinery in `transpiler/declaration.go:853`)
- [ ] Anonymous classes with multiple methods / extending a class
- [ ] Local classes declared inside methods

## 5. Modern Java syntax (Java 10–17+)

- [x] `var` local variable type inference (verified working by e2e suite; was already covered by existing inference)
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
