# Sol checkpoint 194: invocation correctness and CI repairs

Source invocations now retain the inferred return type's declaring-package origin when storing staged results. Primitive argument staging checks visible Java binders that would shadow the emitted Go type. Affine and workflow regression assertions verify complete ordinary or staged invocation dataflow, preserving receiver and argument order, single evaluation, execution context, and collision guards.

Array null checks use direct Go pointer comparison only when declaration and storage proofs certify a physical canonical array pointer. Erased interfaces and unsupported shapes retain Java reference comparison. No performance improvement is claimed without measurement.

The combined candidate completed 404 checks: 402 exited successfully and two recorded baseline NestedSignature assertion failures remained unchanged. There were 3189 passing native parent executions, 84 direct Java/Go observations, the original URI129 parity parent, the full 849-parent runtime race suite, symbol23/parser10 race checks, and all six default/F1 build, race-compilation and lint profiles. The coordinator rechecked all source files, receipts, raw streams, generated trees, observations and owned process-group absence.

The valid CI applications supply24 fresh JVM and24 Go observations. The unchanged Netty reducers supply18 JVM and18 Go observations. Status, stdout, stderr and declared files match. The existing invalid Scope Java fixture remains a parser test; a separate legal field-shadow application provides the additional behavioral check. No invalid Java oracle was accepted as parity.

Hosted checkpoint193 CI had nine new failing compiler parents; this checkpoint repairs and tests those cases. Overall hosted CI remains pending for the new commit, and two pre-existing local assertion failures remain explicit. Full Netty1220 conversion still exceeds the300-second build limit. Full Gson112 and Supplier28 acceptance, performance claims and round completion remain open. Additional verified private Map and stream prerequisites will be integrated separately.
