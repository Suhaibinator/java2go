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

* [ ] Enums (advanced features)
* [ ] Inheritance / polymorphism
    * [ ] Extends / implements
    * [ ] Abstract classes / methods
    * [ ] Interface embedding
* [ ] Decorators / annotations (beyond passthrough as comments and optional exclusion)
* [ ] Anything that checks `instanceof`
* [ ] Types for lambda expressions

## Generics support

Java2go supports Go 1.18+ generics for many common Java patterns:

* Generic classes (e.g. `class Box<T>`) become parameterized Go types (e.g. `type Box[T any] struct { ... }`).
* Generic constructors and `new` calls support explicit type arguments and the diamond operator (`<>`) when the expected type is known from a local variable declaration.
* Nested generic types are handled (e.g. `Map<String, List<Integer>>`).
* Static generic methods are emitted as generic Go functions.
* Instance generic methods are modeled via generated helper types (since Go methods can’t declare their own type parameters).

Current limitations:

* Type parameter bounds (e.g. `<T extends Number>`) are converted into Go constraints, but complex cases may still be approximated.
* Wildcards and variance (`?`, `? extends`, `? super`) are approximated (often as `any`).
* Generic interfaces are not fully modeled as parameterized Go interfaces yet.

## Usage

* Clone the repo

* `go build` to build the java2go binary

* `./java2go <files>` to parse a list of files or directories

## Options

* `-w` writes the files directly to their corresponding `.go` files, instead of `stdout`

* `-output` specifies an alternate directory for the generated files. Defaults to putting them next to their source files by default

* `-q` prevents the outputs of the parsed files from appearing on `stdout`, if not being written

* `-ast` pretty-prints the generated ast, in addition to any other options

* `-symbols` (WIP) controls whether the parser uses internal symbol tables to handle things such as name collistions, resulting in better code generation at the cost of increased parser complexity (default: true)

* `-sync` parses the files in sequential order, instead of in parallel

* `-exclude-annotations` specifies a list of annotations on methods and fields that will exclude them from the generated code
