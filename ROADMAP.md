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
- [x] Collections: `ArrayList`, `LinkedList`, `HashMap`, `TreeMap`, `HashSet`, `TreeSet` — `stdjava.List/Map/Set` (insertion-ordered maps for deterministic iteration; Tree* sorted-iteration approximated, documented)
- [x] `java.util.Collections` utilities (`sort`, `reverse`, `max`, `min`, ...) + `Arrays.asList/sort/toString`
- [x] Iterators / `Iterable` protocol — enhanced-for ranges over collection `.Slice()` views
- [x] Wire `java.util.Optional` to `stdjava.Optional[T]` (of/empty/ofNullable/get/isPresent/orElse/map/ifPresent; lambda-param inference for `map` is a §2-remainder follow-up)
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
- [x] Anonymous classes implementing a SAM interface — lowered to the existing `FuncAdapter` machinery with enclosing locals as ordinary closure captures
- [x] Anonymous classes with multiple methods / extending a class — synthesized uniquely-named file-scoped structs with captured locals as fields
- [x] Local classes declared inside methods — hoisted to file scope with captures as fields

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

- [x] `synchronized` blocks/methods → identity-keyed monitors (`stdjava.MonitorEnter/MonitorExit`), race-tested
- [x] `Thread` / `Runnable` → goroutines (lambda/method-ref form; `extends Thread` and anonymous `Runnable` are follow-ups in task #11)
- [x] `volatile` → documented limitation (doc comment on the field; atomic rewrite of every access site deliberately out of scope)
- [x] `java.util.concurrent.*` basics — `AtomicInteger/Long/Boolean` (sync/atomic), `ConcurrentHashMap`, `ExecutorService` worker pool
- [x] `wait`/`notify`/`notifyAll` → `sync.Cond` over the identity-monitor mutex (atomic release/re-acquire matches Java; timed `wait(millis)` documented as unsupported)

## 8. Out of scope / decide explicitly

- [ ] Reflection (`getClass()`, `Class.forName()`, `java.lang.reflect.*`) — decide whether to support a subset or document as unsupported
- [ ] Module system (`module-info.java`, Java 9+)
- [ ] Class loading semantics / custom class loaders
