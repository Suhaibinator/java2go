# Canonical Writer and exception constructor checkpoint19

Java-facing Writer defaults and virtual dispatch accept canonical String references while preserving UTF-16 units, null behavior, range checks, synchronization and buffer reuse. AssertionError and IllegalArgumentException constructors preserve message references and tested cause and callback behavior. Native Go entry points remain covered.

Actual failing regressions precede the general fixes. The integrated original IO10 plus focused/runtime controls passed 23 parents, independently audited with 507 checks. Publication starts from Git base `d753d5f1b6a730f15068d504974f017cd121b8c0` and changes exactly 14 code/test paths. Original Java programs and JVM expectations are preserved; four harness migrations compare non-null canonical String UTF-16 units.

Final fresh publication verification passes 73 parents and 141 test keys, including 23 generated race tests and nine race-enabled compilation-only fixtures, with no failures or skips. All 32 generated fixtures, 64 nested receipts and four PATH restoration receipts are independently verified. Pinned golangci-lint 2.14.0 passes stdjava, transpiler and astutil; four strengthened constructor test checks also pass 18 test keys. The lint correction changes only four test return-value sites, adding receiver identity checks and explicit failures when expected exceptions are not thrown.

Earlier transport and lint failures remain preserved. The final gate was rerun after the test correction, and source/tool integrity and process cleanup were verified. This checkpoint is scoped compatibility evidence; cumulative CI, full Gson and the separate NPE, Map and dollar candidates remain pending. Fresh JVM calls remain in unchanged helpers, but separate raw JVM subprocess streams are not retained for every IO fixture.

Tested 1,943-file source manifest: `0e87c93cea12f0628bcecb5b92adb3f2fe3eb1b267fcda97fb27da6119a7177b`.
Final terminal receipt: `61e4ddac51dc97a05c12f0c449f7fd49604fba5f696c08b487149e57ed1fce90`.
Independent audit: `801b1d6ca5ec42e25644f8fef48e6e6f2b64bd3932eb3332bd3f0e213534bae9`.
Focused runtime/lint receipt: `6e201b6b2e0f849146be69aac493029d71959bec54cf3a3d0dd4456118d5d6e7`.
