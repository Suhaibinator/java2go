# Java2go
## About

Java2go is a transpiler that automatically converts Java source code to compatible Go code

It does this through several steps:

1. Parse the java source code into a [`tree-sitter`](https://github.com/smacker/go-tree-sitter) AST

2. Convert that AST into Golang's own internal [AST representation](https://pkg.go.dev/go/ast)

3. Use Golang's builtin [AST printer](https://pkg.go.dev/go/printer) to print out the generated code

## Issues

Note: Java2go is still in development, and as such, please expect many bugs

Java2go supports a growing subset of Java 21, including abstract classes,
nominal casts and `instanceof`, and target-typed lambdas. It does not yet support
the complete Java language or standard library. Remaining work includes
annotations beyond comments/exclusion, general reflection and serialization,
full Unicode String fidelity, and remaining generic/standard-library edge cases.
See [ROADMAP.md](ROADMAP.md) and the application parity corpus for coverage.

## Enum support

Java2go provides comprehensive enum support, converting Java enums to Go structs with singleton instances:

* Basic enum constants become pointer variables to struct instances
* Enums with fields and constructors are fully supported
* Standard enum methods are generated:
  * `EnumNameValues()` returns all enum constants as a slice
  * `EnumNameValueOf(name string)` converts a string to an enum constant
  * `Name()` and `Ordinal()` accessors
  * `CompareTo(other)` for comparing enum constants
* Enums implementing interfaces embed those interfaces in the generated struct
* Constant-specific class bodies (method overrides per constant) are supported via dispatch wrappers
* Abstract methods in enums generate wrappers that panic for unimplemented constants

## Generics support

Java2go supports Go 1.18+ generics for many common Java patterns:

* Generic classes (e.g. `class Box<T>`) become parameterized Go types (e.g. `type Box[T any] struct { ... }`).
* Java type parameter bounds (e.g. `<T extends Number & Comparable<T>>`) are converted into Go constraint expressions on structs, functions, and generated helpers/constructors. `Number` uses the `stdjava.JavaNumber` accessor interface. Built-in `Comparable<T>` has an erased Go representation with Java type metadata retained for checked dispatch.
* Generic constructors and `new` calls support explicit type arguments and the diamond operator (`<>`) when the expected type is known from a local variable declaration.
* Nested generic types are handled (e.g. `Map<String, List<Integer>>`).
* Static generic methods are emitted as generic Go functions.
* Instance generic methods are modeled via generated helper types (since Go methods can’t declare their own type parameters).

Current limitations:

* Complex bound combinations or wildcard/variance semantics may still be approximated when translated into Go constraints.
* Wildcards and variance (`?`, `? extends`, `? super`) are approximated (often as `any`).
* Generic interfaces are emitted as parameterized Go interfaces, including constraints derived from Java bounds.

## Boxed values and generated Go compatibility

All eight Java wrappers (`Boolean`, `Byte`, `Short`, `Character`, `Integer`,
`Long`, `Float`, and `Double`) use nullable, immutable `*stdjava` objects.
Primitive Java values remain Go scalars. Boxing preserves the supported Java
cache identities; explicit wrapper constructors allocate fresh objects.
Unboxing null throws `NullPointerException`. Wrapper reference comparisons,
value equality, casts, numeric accessors, and collection keys keep their
distinct Java semantics, including NaNs and signed zero.

For example, Java `List<Integer>` becomes `*stdjava.List[*stdjava.Integer]`.
Handwritten Go callers can use `stdjava.BoxInteger(42)` and
`stdjava.UnboxInteger(value)`. Primitive streams and primitive Optionals keep
scalar elements; reference streams and generic Optionals use wrapper objects.

Generated Java varargs parameters use the same array wrappers as ordinary array
parameters: `int...` becomes `*stdjava.PrimitiveArray[int32]`, and reference
varargs use `*stdjava.ReferenceArray`. Calls with an existing array preserve its
identity and nullness; expanded calls allocate a fresh array with runtime
component checks.

This changes the generated Go API. Regenerate the complete output tree and use
the matching `stdjava` runtime revision, then adapt handwritten Go callers to
the wrapper and array APIs. There is no legacy scalar-wrapper mode. Arbitrary
user-defined collection `equals`/`hashCode`, general reflection, serialization,
and unrelated library expansion remain separate work.

## Usage

* Clone the repo

* `go build ./cmd/java2go` to build the java2go binary

* `./java2go <files>` to parse a list of files or directories (or run directly with `go run ./cmd/java2go <files>`)

## Application parity tests

The project includes deterministic, multi-package Java applications that are
compiled and run with a real JDK, transpiled as complete source trees, compiled
as Go, and compared byte for byte:

```sh
go test ./e2e -run '^TestApplicationParity$' -v
```

Passing applications enforce regressions immediately. Known-gap applications
must reproduce a pinned failure and become strict TDD targets with:

```sh
JAVA2GO_PARITY_STRICT=1 go test ./e2e -run '^TestApplicationParity$' -v
```

Two additional live-oracle suites protect the existing corpus. The first runs
all programs below `testfiles/e2e`; the second automatically discovers older
runnable Java programs elsewhere under `testfiles`. Both compile and run Java
and generated Go in isolated directories, then compare exit status, stdout, and
stderr exactly:

```sh
go test ./e2e -run '^(TestE2EPrograms|TestLegacyApplicationParity)$' -v
```

CI sets `JAVA2GO_PARITY_STRICT=1`, so a known-gap application cannot be accepted
on the protected test path.

See [`testfiles/applications/README.md`](testfiles/applications/README.md) for
the fixture contract, current application matrix, and promotion workflow.

CPU-intensive fixtures also carry `benchmark.json`. The benchmark harness
builds both implementations outside the timer, performs an untimed validation
run, then verifies the exact parity oracle after every measured process:

```sh
go test ./e2e -run '^$' -bench '^BenchmarkApplicationPerformance$' -benchtime=1x -count=3
```

Results include runtime startup and shutdown. The workloads are deliberately
large enough to keep that overhead from dominating; use the reported `ns/run`
metric to compare individual executions when a fixture batches multiple runs.
The checked-in workloads are calibrated for roughly ten-second-or-longer Java
runs on the reference development host while staying well below 8 GiB peak RSS.
Those values guide workload sizing rather than acting as portable test limits.

## Options

* `-w` writes the files directly to their corresponding `.go` files, instead of `stdout`

* `-output` specifies an alternate directory for the generated files. Defaults to putting them next to their source files by default

* `-q` prevents the outputs of the parsed files from appearing on `stdout`, if not being written

* `-ast` pretty-prints the generated ast, in addition to any other options

* `-symbols` (WIP) controls whether the parser uses internal symbol tables to handle things such as name collistions, resulting in better code generation at the cost of increased parser complexity (default: true)

* `-sync` parses the files in sequential order, instead of in parallel

* `-exclude-annotations` specifies a list of annotations on methods and fields that will exclude them from the generated code

* `-init-go-mod` creates a `go.mod` file in the output directory when writing files (`-w`)

* `-module` sets the module path used by `-init-go-mod` (default: `generated`)
  * When Java packages share that prefix (for example module `com/acme` with package `com.acme.app`), generated files are written module-relative (for example `app/MainApp.go`)
