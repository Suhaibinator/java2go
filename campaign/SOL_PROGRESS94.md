# Sol progress94: compiler source inventories

Class-to-file ownership lookup and static-import parsing now reuse exact structural facts through a conversion. Static imports preserve declaration order, fresh graph lifetimes, and caller-owned returned slices. Symbol resolution, lexical permissions and intrinsic admission still run normally.

Coordinator verification passed all114 compiler records with no failures or skips and pinned golangci-lint2.14 with zero issues. Independent review covers the seven-file delta, eleven JVM comparisons and three strict rejection pairs. Frozen allocation ratios improved21.08 to1.14 for ownership and31.99 to1.01 for static imports, both below the unchanged limit6.

Full Netty remains unaccepted. Its latest strict run before the static-import change timed out at300.202944 seconds; an updated full run is required. Commons27, genuine Gson118 and full accumulated CI remain open. Checkpoint18 is still the latest accepted full round.

The separate foundation integration was rejected by coordinator crossing checks: a misplaced test and two compiler allocation regressions. Test relocation is sealed; the measured recursive volatile-declaration scans are being repaired. These changes are excluded from this milestone.

Resume the Sol-only campaign with the same-source foundation gates, reflection linkage and bridges, generic bounds, source collection contracts, Gson signature/core lowering and Maven graph completeness. Preserve original inputs, expectations, failures and user files; commit and push verified milestones.
