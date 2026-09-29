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


### Checkpoint19 continued adversarial review

Private repairs now pass six focused source-text dispatch controls and fifteen static-import/timer controls. Independent review identified an additional canonical Object anonymous-registration case. The caller-import String-shadow control is JVM-valid and now reproduces a Go build failure; both require resolution before promotion. These focused results are not new application acceptance. Production remains checkpoint18.

The unchanged sixteen-case String workflow now matches fifteen normal and nine race observations, including argument evaluation order and bounds exceptions. It still fails constructing a legal String containing isolated UTF16 surrogates; the final race observation is unreached. Two smaller JVM-valid tests reproduce the runtime rejection. An initially incorrect authored hash expectation was corrected using the JVM result, with the invalid attempt preserved. The existing immutable JavaString core is suitable storage, but activating its compiler/runtime ABI requires coordinated work.

The timer driver passes nine semantic groups and three race observations. Its timing pilot is UNQUALIFIED: Java batches are too short and fail the declared stability check, and sampled Go medians are below the duration threshold. No Java-versus-Go speed claim is supported. A separately versioned batching proposal is under preparation; existing workload sources, thresholds and results remain unchanged. Private attempt archives and exact resume ownership are recorded in campaign-state.json.


### Checkpoint19 integration and canonical String preparation

The private native-String repairs now pass seven source-text controls plus 53 existing neighbors, and 43 static-import/declaration-result controls. The combined source caught a stale helper call during compilation, then an old Go-only charAt test expecting a native slice panic. The helper call is repaired; the charAt expectation now follows the frozen JDK21 exception contract and retains every prior input and control. Seven focused race tests pass. A fresh combined candidate passes build, exact lint2.14 and noncompiler tests; its compiler shard has exposed an assertion tied to the old compareTo expression shape. A narrowly reviewed test update preserves the Java source and checks receiver-before-argument staging, the null check, intrinsic lowering and interface-bound method counts. Full verification and final application endpoints remain pending. No private implementation has been promoted.

The separate canonical String candidate combines compiler lowering with focused charset, container and registered source-text helper repairs. Compiler-only compilation succeeds, but this is not generated application parity. Its inventory still lists 27 missing helper symbols and additional source, enum, Throwable, default Object and runtime-family bridges. New char-array and host-output helpers are being developed through JVM-first tests. The unchanged isolated-surrogate workflow and full Gson challenge remain required failures.

The separately versioned fixed-cycle timing driver passes independent semantic/model checks and its duration/stability pilot. Unequal repetition counts make raw batch times incomparable. JVM compilation/GC diagnostics and fresh uninstrumented measurement forks remain pending; no Go-over-Java CPU advantage is claimed. Lazy literal-cache work remains a design proposal. Private failure histories, helper red/green evidence, frozen integration inputs and performance audits are retained in the ignored checkpoint archives, with hashes and resume ownership in the tracked ledger.

### Checkpoint 19: full native regression gate and generic collection follow-up

The frozen native v6 integration passed 1,901 unit pass records, 55 strict parity tests, 73 other end-to-end tests, the prior fuzz inventory and known skips, lint 2.14, and the five accumulated passing applications (45 normal pairs and 100 race stress pairs). Full Gson revealed a new strict-transpilation regression at `newImmutableList(builder.factories)`; this prevents promotion despite the historical successes. The complete failed snapshot and raw evidence are preserved in ignored checkpoint19/private-attempts-v7 with a verified archive manifest.

A focused repair preserves method generic binders during JDK collection conversion; its two original JVM-valid failing tests and two invariant/source-shadow controls pass strict transpilation and race execution. The untouched full Gson application then advances to `newImmutableList(builder.reflectionFilters)` and fails because ArrayDeque ancestry to Collection is absent. All nine JVM observations are stable; Go compilation and execution remain unreached. Independent review also identified a potential bounded-binder overload issue, now assigned a JVM-first control. Neither candidate is promoted.

Canonical String work remains private: boxed, char-array, and UTF-8 host helpers passed their focused JVM/race contracts and are assembled in a new unrun snapshot. Search/prefix helpers and charset overloads await focused execution. Source text and Throwable bridges remain prerequisites. Performance agents found and are correcting evidence-validation gaps in the prepared measurement harness; no Java-versus-Go speed advantage is claimed.

Resume with the generic-bound control and ArrayDeque ancestry repair, then a fresh full integration gate and independently audited supplemental applications. Full Gson remains the required failing challenge; no stage advancement or round acceptance is recorded.

The bounded-binder review control subsequently confirmed an introduced regression: JVM and native43 select the Object overload, while the binder candidate selects the numeric generic overload and fails generated Go race compilation. The control is frozen and a general bound-applicability repair is required before integration.

### Checkpoint 19: canonical String integration and generic-call repairs

Nine unchanged JVM/generated-race controls now pass for canonical String storage, identity, isolated UTF16, nulls, switches, interning, and source text callbacks. The original larger workflow remains required: all sixteen JVM/model observations passed, but generated Go stops at four Integer.parseInt calls that still expect native strings. No normal or race application pairs were reached. UTF16 numeric parsing and exact exception-message storage are the next prerequisites. Three charset-overload tests, four qualified-charset tests, seven search helpers, and trim/strip/isBlank passed their respective focused JVM/race gates; the whitespace comparison covers every Unicode code point and a 679-line content/identity transcript. These component results do not establish full ABI acceptance.

Native generic repair V3 passes twelve focused controls, including numeric bounds, declaring-owner qualification, repeated invariant parameters, and upper wildcard capture. A further JVM-first lower-wildcard control reveals incorrect overload selection and remains active for V4. Objects.requireNonNull passes its original four contracts; the added String-bounded message/null-call control exposed two further defects. Their private fixes are entering a combined nineteen-test gate. Full Gson remains unchanged and required; its most recent strict failure is the Objects import, after the ArrayDeque ancestry fix advanced translation.

Performance validation now rejects malformed metrics and mismatched evidence in all 38 Python controls. JFR preparation matched three frozen models, but capture/logging probes exposed startup output pollution; original artifacts are preserved, and official HotSpot source guides a separate logging revision. The workload output parser remains strict, and there is no steady-state or speed claim. New runs use a verified process supervisor with unconditional cleanup receipts.

This checkpoint preserves experiments and exact blockers, not a production promotion. After the generic fixes are combined, rerun the unchanged Gson application, full accumulated regression shards, and supplemental endpoints from a fresh frozen snapshot.

### Usage-limit resume checkpoint

Agent execution hit the account usage limit. No matching active campaign test process was observed; the final collection baseline grant had not produced execution evidence and remains unrun. A verified 9,505-file archive preserves current source candidates, regressions, runtime evidence, and the exact resume inventory under the ignored checkpoint19/usage-resume-20260929 directory.

Since the prior checkpoint, V4 retained all twelve generic controls but exposed the physical Collection<? super T> / Iterable[any] incompatibility. A dedicated agent prepared a stateful collection baseline; no runtime workaround was accepted. The Objects/native-string conversion is staged in combined-v2, whose nineteen-test gate is unrun. Throwable helper tests passed five JVM groups and 38 native compatibility tests; compiler lowering is still separate. Integer.parseInt has valid JVM oracles and recorded compiler/runtime reds, with a partial candidate preserved for inspection. JFR logging V3 passed its strict flag probe; diagnostic capture is still unrun.

Resume from campaign-state.json. Full Gson and the original sixteen-case String workflow remain required and failing; no new production code or round acceptance is claimed.

### Checkpoint19: resumed focused verification and new adversarial failures

The combined nineteen-test generic/Objects gate now passes without failures or skips. The original full Gson application retains nine stable JVM observations and advances to a new strict-transpilation failure at Math.toIntExact. Its focused boundary and conversion tests pass, while a source Math shadowing control exposes an import-table defect: static wildcard imports incorrectly hide a source type. The wildcard import-table correction now passes the three Math tests and two explicit-import compatibility controls. A separate static nested wildcard type import remains a baseline and candidate failure; it is retained explicitly.

The erased-collection candidate matches a thirteen-line stateful JVM transcript and passes all thirteen original generic controls, including the lower wildcard. A fresh Sol challenge has nine stable JVM observations but fails generated Go compilation on raw List signatures and iterator calls. It remains active; broader cast timing, aliasing, array stores, and collection regressions must pass before integration.

Canonical Integer.parseInt now passes UTF16 value/message oracles, strict static-import lowering, and source-shadow controls. The unchanged sixteen-case String workflow builds and passes thirteen normal and seven race observations before a Throwable boundary failure. The generated getter returns native text, and the legacy constructor also mistakes a canonical String argument for a cause. Constructor and virtual getter lowering require coordinated repairs; later bounds and isolated-surrogate cases were not reached.

JFR capture succeeds with all fifty-two markers and twenty-four model rows. Independent review finds kernel compilation completed during warmup, but other compiler events and a safepoint overlap measured batch brackets. Steady state and a Go-over-Java speed advantage remain unproven. The performance agents are preparing a separately versioned, bounded diagnostic for the actual formatting and marker paths.

Verified evidence is retained in ignored checkpoint19/private-attempts-v13 through v15, with hashes in the tracked ledger. These are focused milestones, not production promotion or round acceptance. Full Gson, the original String workflow, accumulated regression gates, and supplemental applications remain required.

### Checkpoint19: restored Gson transpilation and retained boundary failures

The unchanged full Gson application passes strict transpilation in 139.361 seconds after the Math and generic repairs. All nine JVM observations remain valid. Generated Go still fails on the missing collection, generic, reflection, and date/time facilities in the first diagnostic batch. An independent exact-baseline comparator rejects two shifted line numbers; a single new source-text registration explains both shifts, while the affected statements and ordered diagnostic messages remain unchanged. The original baseline is preserved, and truncated diagnostics do not rule out later failures.

The collection candidate now passes nine normal and three race observations against the Sol auxiliary oracle, plus all fourteen earlier focused controls. Fresh controls expose raw removal, source Object/nested consumer projection, and an Iterable method-boundary incompatibility. The Throwable compiler candidate passes storage and virtual callback controls but fails the null-cause rejection diagnostic; separate runtime and constructor-typing repairs are underway. Each failure retains its unchanged JVM oracle.

Seven Math/import controls pass, including the previously failing static nested type and field/type namespace cases. A combined twenty-six-test native gate is prepared. Another source-initialization control remains unrun. The performance prelude preparation has independently matched eighty-four model rows across three seeds; capture remains pending. Verified terminal evidence through private-attempts-v18 is preserved. These milestones do not promote private implementation or complete the active round.

Canonical integration component update: see `campaign/STRING_CANDIDATE_CHECKPOINT19.md` and `active_resume.canonical_lint_component`. Exact final1933 lint and six focused tests are independently green; originalfull22/Gson remain required. NumberEnum3cfd remains on its separate component branch.
