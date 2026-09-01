# Workflow engine parity fixture

This fixture is a deterministic, self-contained workflow scheduler intended for
Java-to-Go differential testing. It models a small nightly synchronization
application rather than a collection of isolated language snippets.

The application exercises:

- thirteen source files in four Java packages;
- generic classes and generic functional interfaces;
- concrete implementations of a generic rule interface;
- enums with fields, constructors, standard methods, and state comparisons;
- `List` and `Map` construction, mutation, lookup, and enhanced `for` loops;
- a stable selection-sort scheduling algorithm;
- dependency resolution over multiple fixed-point passes;
- success, retry, permanent failure, policy rejection, missing dependency,
  failed-upstream, and cycle state transitions;
- a typed lambda, string operations, nested generic types, integer overflow,
  counters, checksums, and deterministic report assembly.

## Java reference run

From the repository root:

```sh
classes_dir=$(mktemp -d)
find testfiles/applications/workflow_engine/src -name '*.java' -print \
  | sort \
  | xargs javac -encoding UTF-8 -d "$classes_dir"
java -cp "$classes_dir" parity.workflow.app.WorkflowApplication
```

The process accepts no input. Its standard output must match
`expected.stdout` byte for byte, including the final newline.

## Current parity status

This fixture has full parity. Strict transpilation succeeds, every generated Go
package compiles, the application exits successfully, and its stdout/stderr
match the Java oracle byte for byte.

The fixture remains a regression target for generic interface implementation,
generic and cross-package chained method resolution, typed SAM lambda bodies,
String intrinsics, Java-width enum ordinals, and collision-safe generated import
aliases.
