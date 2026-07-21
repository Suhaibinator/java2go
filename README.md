# Java2go
## About

Java2go is a transpiler that automatically converts Java source code to compatible Go code

It does this through several steps:

1. Parse the java source code into a [`tree-sitter`](https://github.com/smacker/go-tree-sitter) AST

2. Convert that AST into Golang's own internal [AST representation](https://pkg.go.dev/go/ast)

3. Use Golang's builtin [AST printer](https://pkg.go.dev/go/printer) to print out the generated code

## Issues

Note: Java2go is still in development, and as such, please expect many bugs

Currently, the following features are not implemented (or only partially implemented):

* [x] Abstract classes (abstract methods in enums are supported)
* [ ] Decorators / annotations (beyond passthrough as comments and optional exclusion)
* [ ] Anything that checks `instanceof`
* [ ] Types for lambda expressions

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
* Java type parameter bounds (e.g. `<T extends Number & Comparable<T>>`) are converted into Go constraint expressions on structs, functions, and generated helpers/constructors.
* Generic constructors and `new` calls support explicit type arguments and the diamond operator (`<>`) when the expected type is known from a local variable declaration.
* Nested generic types are handled (e.g. `Map<String, List<Integer>>`).
* Static generic methods are emitted as generic Go functions.
* Instance generic methods are modeled via generated helper types (since Go methods can’t declare their own type parameters).

Current limitations:

* Complex bound combinations or wildcard/variance semantics may still be approximated when translated into Go constraints.
* Wildcards and variance (`?`, `? extends`, `? super`) are approximated (often as `any`).
* Generic interfaces are emitted as parameterized Go interfaces, including constraints derived from Java bounds.

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

See [`testfiles/applications/README.md`](testfiles/applications/README.md) for
the fixture contract, current application matrix, and promotion workflow.

CPU-intensive fixtures also carry `benchmark.json`. The benchmark harness
builds both implementations outside the timer, performs an untimed validation
run, then verifies the exact parity oracle after every measured process:

```sh
go test ./e2e -run '^$' -bench '^BenchmarkApplicationPerformance$' -benchtime=1x -count=5
```

Results include runtime startup and shutdown. The workloads are deliberately
large enough to keep that overhead from dominating; use the reported `ns/run`
metric to compare individual executions when a fixture batches multiple runs.

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
