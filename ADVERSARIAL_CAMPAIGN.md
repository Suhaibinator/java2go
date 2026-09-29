# Continuous adversarial campaign

Status: round 01 accepted; round 02 remains active. Checkpoint18 is the latest independently verified compatibility milestone. All 1,857 unit records, 55 strict application-parity records and 73 other end-to-end records pass without added failures or skips. Fuzz replay passes with the same 8 known-open skips. Build, exact golangci-lint 2.14.0, tagged campaign-wrapper compilation and vulnerability checking pass. Five accumulated dependency applications pass 45 exact JVM/Go observations and 100 additional race-enabled stress runs.

The unchanged full Gson application remains required and unaccepted: nine JVM observations, zero accepted Go observations. Strict transpilation succeeds under the original build budget; all-package Go compilation still fails on AbstractMap, AbstractSet, ConcurrentMap, an unbound generic T, DateFormat, Enum, Duration and Instant. Selected dependency implementations remain translated in full, with no handwritten dependency replacement or JVM delegation.

Checkpoint18 adds reflection-free source Object text and Integer.toHexString behavior, preserves caller execution through bounded generic text conversion, and emits erased result bridges for specialized source adapters. The unchanged NoMetadata and complete Throwable workflows each match nine JVM and nine race-enabled Go observations. The stateful factory/cache application also matches nine binary-JVM, nine dependency-source-JVM and nine race-enabled Go observations. Runtime map key/value view contracts pass JVM/native-driver checks; general translated-source Iterable and live Entry support remain open.

A cold StringRequireNonNull diagnostic change removes a measured Go boxing-allocation path. The unchanged generated full-scan workloads drop from 8,192/5,146/5,000 allocations to zero in the recorded windows for seeds 17/97/65537; sampled scans drop from 128/120/128 to zero. These are warmed Go allocation measurements, not a universal per-call guarantee or evidence of a CPU/JVM speed advantage. StringCharAt still scans prefixes; its wider representation limits remain open.

The user-reported lint findings are repaired. campaign-state.json records fingerprints, failed attempts, repairs and exact resume points. Earlier checkpoint sections below are historical records. Base revision: c1f1ae45cfe8d5465425efc562a54580add4f3ea. Local checks do not establish hosted Linux execution or cold-cache bootstrap.

Three Sol agents own independent Java applications and JVM-derived oracles. Three Astra agents own compiler, runtime, and dependency/harness implementation. Two dedicated Astra agents investigate generated-code and runtime performance; an additional Astra agent reviews lint and regressions. The coordinator independently verifies promotion. Challenges and upstream implementations may not be weakened.

## Execution contract

Use the installed JDK 21 explicitly; seeds 17, 41, 97 each run three times before freezing. Dependencies and complete selected implementation source graphs are locked and translated, never replaced by handwritten library behavior or a JVM bridge. Unsupported calls, timeouts, compilation failures and changed observations are failures. Java package cycles must be handled by the compiler rather than source reorganization.

Work happens in the managed `adversarial-java` checkout. Existing user changes remain in the original checkout. Agents own disjoint files; coordinate before shared edits. Transpiler processes and scratch directories are isolated because symbol tables are global. Commit and push verified milestones to the authorized origin branch; publish no unrelated changes.

## Teams

| Agent | Model | Ownership |
| --- | --- | --- |
| sol_business | gpt-6-sol, high | testfiles/campaign/round*/business |
| sol_data | gpt-6-sol, high | testfiles/campaign/round*/data |
| sol_concurrent | gpt-6-sol, high | testfiles/campaign/round*/concurrent |
| astra_compiler | gpt-6-astra, high | Compiler, symbols, parsing, project lowering |
| astra_runtime | gpt-6-astra, high | stdjava runtime and coordinated lowering |
| astra_build | gpt-6-astra, high | Dependency bootstrap, campaign runner and harness |
| astra_lint | gpt-6-astra, high | Exact CI lint remediation and regression checks |
| astra_perf_codegen | gpt-6-astra, high | CPU/allocation audit and measurement under performance/codegen |
| astra_perf_runtime | gpt-6-astra, high | Runtime/concurrency/memory audit under performance/runtime |

## State and resumption

`campaign-state.json` records active challenges, dependency pins, baseline, and gates. `.campaign/` contains ignored downloaded artifacts and run logs; locks and fixture sources are tracked. Rebuild the compiler and rerun affected gates whenever implementation changes. Accepted challenges remain permanent regressions. After a passing round, Sol creates fresh challenges. Continue until explicitly stopped or execution/usage limits intervene; never equate corpus parity with universal Java support.

## Verification commands

Run a frozen application with full concurrency acceptance using:

```sh
JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home \
TMPDIR=/private/tmp go run ./cmd/javacampaign \
  -fixture testfiles/campaign/round01/concurrent -race -stress-runs 20
```

The flags apply to any fixture. The runner builds all generated packages and the application with race instrumentation, runs the nine normal pairs, then cycles frozen seeds through twenty additional independent executions against the validated JVM oracle. Every result is recorded, and source/implementation fingerprints must remain unchanged. The tagged campaign CI gate requests both flags for every fixture. Uninstrumented runs remain available for diagnosing the first build failure and do not satisfy concurrency acceptance.

The performance reports distinguish observed regressions, isolated optimization candidates, and demonstrated application speedups. Currently there is no demonstrated whole-application speedup. Source-level compatibility remains the round's promotion gate.

## Milestone checkpoints

Commit and push meaningful milestones to `origin/codex/adversarial-campaign`, as explicitly requested by the user. Keep incomplete repair patches uncommitted when recording a stable checkpoint. A checkpoint may record an active failing challenge; it must state that failure clearly and must not be labeled a completed round. Never force-push milestone history.

### Checkpoint06 follow-up

Five accumulated applications passed 45 JVM/Go pairs and 100 stress runs. Exact lint 2.14.0, strict application parity, other end-to-end tests and fuzz replay passed. The unit gate found an affine helper naming regression, now repaired against the unchanged test. Two concurrent full Gson runs timed out during conversion; a quiet instrumented replay completed in 96.810 seconds under the same 300-second bound. Checkpoint07 reruns the official Gson harness quietly before fresh historical verification. Neither timeout is accepted as parity.


### Verification scheduling

Checkpoint07 exact lint is clean. Its first full unit and strict application-parity gates timed out, including a Java oracle; those runs are failures. Sequential reruns under the original limits now pass: the full unit compiler package took 447.746 seconds and strict parity took 653.655 seconds; other end-to-end, fuzz and all five historical dependency applications also passed. Input hashes remained unchanged. This verifies checkpoint07, while full Gson still fails and round02 remains active. Candidate08 and its focused repairs remain separate from this verified implementation.


### Checkpoint09 repair status

The exact 2.14.0 lint gate passed after removing one unused helper. The immutable full unit gate failed with eight new regressions; later gates did not run. Boxed generic consumption and scope-aware local identifier repairs now pass focused race checks, including cast timing, null unboxing, resource cleanup and generated implicit types. These changes are integrated but not yet a verified milestone. Checkpoint07 (`032d3550`) remains the last fully verified and pushed checkpoint.


### Checkpoint10

Connected generic families now preserve shared storage, nominal casts, boxed consumption, local interface implementations and varargs SAM array checks across the exercised contracts. SQL date/time types preserve their canonical owner and util.Date relationship, including bounded generic method references. Local identifier hygiene respects body requirements and renamed resource cleanup. Passing the render context through execution-name generation prevents repeated family rediscovery.

Independent verification checked raw streams, declared files, frozen source/dependency/resource hashes, all nine repeated JVM oracles per application, translated-source versus binary dependency oracles, and twenty race-enabled stress runs per application. All historical observations matched. A verbose original/checkpoint07 fuzz audit confirms the eight remaining skips are pre-existing. Gson’s Go compilation failure remains an explicit blocker, so this is a checkpoint, not round02 acceptance. Private nested-enum and big-number TDD work continues after this milestone; unmeasured hygiene caching and runtime allocation candidates remain isolated.


### Checkpoints11–14: integrated repairs, full verification pending

Focused JVM comparisons now cover interface member enums and static storage, block-local type visibility, inherited member access across package boundaries, and local generic owners. Unicode handling adds repeated-u eligibility, original-source diagnostic offsets and eligible surrogate-pair literals. This is not complete Java Unicode preprocessing; isolated surrogate string storage remains unsupported.

BigInteger and BigDecimal services implement the exercised constructors, parsing errors, Number conversions, comparison, scale, equality and hashing. AssertionError constructors preserve nullable messages, primitive and Object details, caller execution and exercised cause contracts. The full unit gate exposed a no-detail assertion wrapper regression, repaired by forwarding the original constructor arity. Source subclasses overriding initCause during AssertionError construction remain an explicit unimplemented boundary.

Checkpoint13 passed all 1,672 unit test records with no failures or skips. Its strict application gate found one regression in the unchanged nested math application: relative Outer.Inner names lost their owner across compilation units. A general first-segment binding repair now passes that fixture, cross-file and import JVM comparisons, and lexical/import precedence guards independently. Checkpoint14 reruns the complete accumulated gates against a fresh immutable snapshot. Its runner records independent later failures as well; any failed gate still makes the run fail.

Frozen nullable Function and ConcurrentMap prerequisite probes, and the larger shared-collection-family application, remain pending work toward the same unchanged Gson acceptance test. They cannot substitute for full dependency translation or application parity. Performance candidates remain isolated and unmeasured. The ledger preserves exact red/green artifacts, hashes, ownership and live runner handles; no new milestone is claimed before full historical verification.


### Checkpoint14 verification

All 1,674 unit test records, 55 strict application-parity records and 73 other end-to-end records pass without added failures or skips. Fuzz replay passes with exactly the same eight open skips. Repository build, golangci-lint 2.14.0, tagged campaign-wrapper compilation and govulncheck pass. The independent audit accepts all five historical dependency applications: 45 JVM/Go observation pairs and 100 additional race-enabled stress observations, including raw streams, file inventories and dependency-source JVM oracles.

The first application attempts stopped before execution because the temporary snapshot lacked Git metadata. The repair installed independent metadata pinned to the existing base revision without changing any source hashes; all failed reports are retained. The repeated application gates establish the acceptance above. Cold-cache dependency bootstrap was not repeated, and these local checks do not establish Linux platform compatibility.

Gson strict transpilation completes in 109.875 seconds within the original 300-second limit, but generated Go still fails to build. The first diagnostics now include AbstractMap, AbstractSet, ConcurrentMap, an unbound generic T, DateFormat, Enum, Duration and Instant. Nine stable JVM oracles and zero accepted Go observations are recorded. Round02 remains active. Seven frozen Function prerequisites additionally reproduce invalid Go lowering and a worker callback that observes creator execution; that repair remains private and is not included in this milestone.


### Checkpoint15: Function identity and measured decimal allocation reduction

Function values now retain object identity, nullable references and the invoking Java thread's execution context. Named and inherited implementations use compiler-resolved SAM bridges; Map, stream, collector and Comparator callbacks use the same contract. General source-object storage prevents Go zero-sized allocations from collapsing distinct Java instances, including local, anonymous and default-method carriers. Strict JVM regressions preserve receiver/argument ordering, source name shadows, overloads and allocation identity. Generated Go must be regenerated against this runtime.

Independent verification passes 1,705 unit records, 55 strict application records, 73 other end-to-end records, exact golangci-lint 2.14.0 and repository build. Fuzz replay retains exactly the same eight prior skips. The vulnerability scan passed after a network retry; the initial DNS failure remains recorded. All five historical dependency applications pass 45 JVM/Go observation pairs and 100 race stress observations, independently checked against raw streams, file inventories, source and binary dependency oracles, and frozen hashes.

Equal-scale BigDecimal comparison directly compares immutable coefficients after both null checks. The focused allocation regression changes from six allocations to zero; separate JVM comparisons and repeated allocation trials pass under the declared UTF8 configuration. This is an allocation result, not a demonstrated speedup over Java. The earlier C-locale output-encoding mismatch remains an explicit configuration gap.

The unchanged full Gson challenge completes strict conversion in 119.340 seconds but still fails all-package Go compilation. The first reported gaps remain AbstractMap, AbstractSet, ConcurrentMap, a free generic T, DateFormat, Enum, Duration and Instant. Nine valid JVM oracles yield zero accepted Go observations. Round02 remains active. Raw/wildcard Function conversions, additional Function APIs, Throwable metadata/dispatch, String reference migration and broader collection contracts remain work in progress. Cold-cache bootstrap and Linux verification were not repeated for this local checkpoint.


### Checkpoint17: String joining, nominal text conversion and measured charAt allocations

This checkpoint assembles 45 changed Go files against the frozen integration base. String.join follows the exercised array, expanded-varargs and runtime Iterable paths, including null checks, callback order, caller execution and checked consumption of erased elements. CharSequence assignment and overload selection use nominal Java relationships, inherited interfaces and type-parameter bounds. Typed nulls and source declarations that shadow canonical names retain their own bindings. Enclosing static-field resolution now supplies the declaring owner's receiver type, while nearer locals, parameters and instance fields continue to hide outer static fields.

Concrete java.lang.Throwable allocation preserves identity, message/cause constructor distinctions and initCause state. Explicit Throwable text calls preserve Java virtual dispatch and the caller's execution. For the exercised registered source classes, objects are classified nominally: unrelated methods named Message, Name or Error do not make an object a Throwable or replace its inherited Object text. Default Object text uses Java class identity in those paths; actual source toString overrides still run. Unregistered source leaves remain an open boundary covered by the retained NoMeta probe. getCause values are projected into the declared Throwable reference at initialization, assignment, return and argument boundaries, preserving identity and nulls. These changes retain the original Java expectations; Go emission assertions were updated only where the corrected generated form changed.

The original complete Throwable workflow has independently matched nine JVM and nine race-enabled Go executions with unchanged Java inputs and source hashes. Independent verification passes 1,834 unit records, 55 strict application-parity records and 73 other end-to-end records. Fuzz replay has exactly the prior eight skips. Build, exact golangci-lint 2.14.0, tagged wrapper compilation and govulncheck pass. The independent raw audit confirms five accepted applications, 45 normal observation pairs and 100 additional race-enabled stress pairs, including frozen source/dependency hashes, binary-versus-source dependency JVM oracles, output-file inventories and exact streams.

The StringCharAt change removes the temporary UTF16 buffer in the measured valid helper cases, reducing their allocation count from one to zero while preserving the existing valid UTF16-unit and invalid-index behavior. The helper still scans the prefix and does not repair isolated-surrogate string storage. Three trials per unchanged generated full-scan workload measured Go allocations as follows:

| Workload seed | Before | After |
| --- | ---: | ---: |
| 17 | 16,280 | 8,192 |
| 97 | 10,178 | 5,146 |
| 65537 | 9,888 | 5,000 |

These counts describe Go allocations for the recorded workloads. They do not establish lower CPU time, a whole-application speedup or a speed advantage over Java.

Broader translated-source Iterable support and full Gson remain required work. The full Gson result for this exact frozen snapshot is strict conversion succeeds in 116.067 seconds, followed by all-package Go compilation failure on AbstractMap, AbstractSet, ConcurrentMap, a free generic T, DateFormat, Enum, Duration and Instant; nine JVM observations and zero accepted Go observations; earlier checkpoint diagnostics remain history and must not substitute for the new report. The separate NoMeta probe reproduces `undefined: Integer` and `value.toString undefined` on both baseline and candidate, with nine fresh JVM observations and no accepted Go observations. Its retained compile failures are neither repaired nor waived by the Throwable workflow result. Round02 remains active. Cold-cache bootstrap and hosted Linux execution are not established by these local checks.

### Checkpoint18: source identity, generic adapter bridges and allocation reduction

Named source classes now expose lightweight nominal descriptors without requiring reflection metadata. Inherited Object.toString uses fixed-arity selection and virtual hashCode with the caller execution; the tested source overrides and name shadows retain their bindings. Java type-parameter identities survive Go-name reservations, including bounded text conversions that synchronize. Local hierarchy layout, constructor identity initialization and nominal cast/view IDs now agree. An unchanged historical constructor test and a separate JVM control caught both missing storage and inconsistent local IDs before promotion. Exact JVM numbering of local/anonymous binary names remains a separate gap.

Specialized source adapters now bridge covariant source-reference results to canonical Object in erased dispatch, while source Object shadows, type parameters and primitive restrictions retain their meanings. The original stateful factory/cache application keeps full Commons Lang implementation sources and tests mutable cache identity, null probes, two adapter families, and failed raw casts before state changes. All three seeds and repetitions pass against binary and source dependency JVM oracles and race-enabled Go.

Runtime key/value views retain their backing map and expose both existing internal collection and execution-aware Java iterator protocols. JVM/native-driver checks preserve replacement visibility, exhaustion and structural invalidation timing for TreeMap and HashMap. This is a runtime prerequisite, not general source Iterable or live Entry acceptance. Existing join tests remain unchanged. A separate ternary emission assertion now examines the generated Java method's AST, excluding unrelated type-registration initializers while retaining the original Java and runtime checks.

The StringRequireNonNull diagnostic uses reflection only to name an invalid dynamic type; successful conversion, null handling and exception construction remain unchanged. Ordinary/race semantic gates, formatter and concurrency controls, unchanged generated scans, allocation controls and normal optimized escape diagnostics were independently audited. The unchanged StringReferenceValue allocation control remains positive. No Java CPU or allocation comparison is claimed. A new pure-Java timing driver is currently blocked by missing System.nanoTime lowering; it has no accepted timing result.

Full Gson remains required and red. Next prerequisites include source Iterator/Iterable bridges and consumer cast timing, actual AbstractMap/AbstractSet defaults and live entries, ConcurrentMap operations, generic factory metadata, DateFormat, Enum reflection and java.time. Netty progression remains gated on the current round. Every failed checkpoint18 snapshot is preserved alongside the accepted source and raw verification artifacts under .campaign/checkpoints/.

A new frozen Sol probe confirms that an ordinary Java method named String() can hijack toString conversion: all nine JVM runs pass, while checkpoint18 fails the first callback assertion in all nine generated Go runs. The older checkpoint17 cannot be classified for this probe because its Go build fails on Integer.toHexString. The fixture and JVM-derived observations are retained under campaign/reproducers/capital-string-collision.

The next performance semantic gate also found that generated String.charAt checks a null receiver before evaluating the index, reversing Java side effects and exception precedence. All 16 JVM observations passed; 13 normal and seven race Go observations passed before the ordering failure. Later Go bounds and isolated-surrogate cases were not reached. Optimization and timing remain gated on the full unchanged semantic workflow. Fresh timer/static-import tests additionally expose source-owner and qualified-value binding errors. These checkpoint19 repairs remain private and unaccepted; full Gson stays required and red.
