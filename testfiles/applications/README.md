# Application parity corpus

This corpus verifies whole Java applications, not just individual syntax
features. It requires JDK 21 or newer and compiles fixtures with
`javac --release 21`. For every fixture the test harness:

1. compiles all Java sources with `javac`;
2. runs the declared Java entry point in a deterministic environment;
3. requires its stdout and stderr to match the checked-in oracle exactly;
4. transpiles the complete source tree in strict mode;
5. compiles every generated Go package and a generated entry-point driver; and
6. runs the Go binary and compares its exit code, stdout, and stderr with Java.

Run the corpus with:

```sh
go test ./e2e -run '^TestApplicationParity$' -v
```

To use every known gap as a failing TDD target, enable strict parity mode:

```sh
JAVA2GO_PARITY_STRICT=1 go test ./e2e -run '^TestApplicationParity$' -v
```

## Current applications

| Fixture | Domain and coverage | Status |
| --- | --- | --- |
| `existing_full_program` | Existing multi-package application reused in place: interfaces, inheritance, generics, collections, exceptions, imports, and cross-package calls | Passing |
| `routing_engine` | Cross-package graph model, relaxation algorithm, interface dispatch, arrays, stable tie-breaking, unreachable routes, checksum | Passing |
| `analytics_pipeline` | Parsing, validation, bounded generics, collections, scoring, aggregation, stable ranking, rejections, checksum | Passing |
| `workflow_engine` | Generic workflow scheduler, rules, enums, collections, sorting, retries, failures, dependency cycles, histories, overflow | Passing |
| `side_effect_semantics` | Short-circuiting, nested ternaries, receiver/argument order, null invocation timing, static qualifiers, compound assignment, `finally`, and recursive side effects | Passing |
| `numerical_kernels` | Floating-point recurrences, blocked dense matrices, iterative stencils, cache locality, allocation, numerical checksums | Passing + benchmark |
| `allocation_gc_pressure` | Short-lived object/array churn, retained cyclic graphs, cohort rotation, traversal, mutation, reclamation pressure | Passing + benchmark |
| `integer_branch_search` | Constraint search, prime sieve, branch-skewed pointer chase, bit operations, recursion, large integer arrays | Passing + benchmark |
| `finally_loop_control_gap` | `break`/`continue` through `finally`, observable side-effect order, and loop-transfer resumption | Known gap (`go_compile`) |
| `array_assignment_timing_gap` | Null-array assignment evaluation order, index/RHS side effects, and exception identity | Known gap (`go_run`) |
| `constructor_nullable_field_gap` | Constructor-time virtual dispatch observing Java's pre-initializer null field value | Known gap (`output`) |
| `recursive_object_model` | Mutual recursion, recursive generic graphs, constructor dispatch, hiding, and inherited interface defaults | Passing |
| `static_method_hiding_gap` | Parent/child static-method hiding and declaring-class selection | Passing |
| `local_class_recursion_gap` | Capturing local class with a recursive instance method | Known gap (`go_compile`) |
| `anonymous_class_recursion_gap` | Recursive override in a synthesized anonymous subclass | Known gap (`go_compile`) |

Known-gap fixtures are not skipped. In normal mode they must still have valid
Java behavior and must fail at the exact declared stage for a pinned diagnostic
substring. A different failure fails the test, and an unexpected parity success
also fails with instructions to promote the fixture. This prevents a new bug
from hiding behind an old known-gap label.

## Fixture contract

Each direct child directory contains:

```text
<fixture>/
  fixture.json
  expected.stdout
  expected.stderr      # optional; absence means stderr must be empty
  README.md
  src/**/*.java
```

An application already stored elsewhere under `testfiles` can be exercised
without duplicating or modifying its sources by omitting `src` and setting a
clean relative `source_root` in `fixture.json`, for example:

```json
{
  "main_class": "com.acme.app.MainApp",
  "module_path": "com/acme",
  "source_root": "../../full_program",
  "status": "passing"
}
```

A passing manifest contains:

```json
{
  "main_class": "parity.example.app.ExampleApplication",
  "module_path": "parity/example",
  "status": "passing"
}
```

A known-gap manifest also contains `known_gap`,
`expected_failure_stage`, and `expected_failure_contains`. Supported failure
stages are `transpile`, `go_compile`, `go_run`, and `output`.

## Adding a TDD application

1. Choose a deterministic, self-contained domain with enough output to expose
   meaningful state transitions and ordering.
2. Avoid clocks, randomness, network access, external files, stdin, environment
   input, locale-dependent formatting, and non-JDK dependencies.
3. Put all sources below `src`, add the manifest, and record the exact JDK output
   (including the final newline) in `expected.stdout`.
4. Start with `status: passing` and run the focused test. If a real translator
   gap blocks parity, change it to `known_gap` and pin the earliest stable stage
   and a specific diagnostic substring. Document that gap in the fixture README.
5. Fix the translator under the strict-parity test. Once the application passes,
   promote the manifest to `passing` and remove all known-gap fields.

The harness intentionally leaves generated source in temporary test directories;
checked-in fixtures contain Java and behavior oracles only.

## Performance benchmarks

A fixture becomes a benchmark by adding a strict `benchmark.json` marker:

```json
{
  "category": "floating_point_matrix",
  "description": "Deterministic floating-point and matrix workload",
  "iterations": 1
}
```

`iterations` is the number of fresh application processes in one benchmark
operation. Compilation and transpilation are excluded from timing. Both Java
and generated Go receive one untimed warm-up, and every timed execution must
still match `expected.stdout` and `expected.stderr` byte for byte.

The checked-in performance fixtures are calibrated to run Java for roughly ten
seconds or longer on the reference development host while remaining well below
8 GiB of peak resident memory. These are workload-sizing targets rather than
portable pass/fail thresholds: host speed, JVM version, and runtime tuning all
affect them. Scale duration with bounded repeated computation rather than an
ever-growing live dataset, and remeasure runtime and peak RSS after changes.

Run one measured operation for every benchmark and repeat it for independent
samples with:

```sh
go test ./e2e -run '^$' -bench '^BenchmarkApplicationPerformance$' -benchtime=1x -count=3
```

The normal `ns/op` value measures the configured batch. The additional
`ns/run` metric normalizes it to one application process. These are end-to-end
application timings, so process/runtime startup and shutdown are intentionally
included.
