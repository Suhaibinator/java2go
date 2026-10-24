# Sol checkpoint 233: bounded raw families and compiler inventories

A valid Java workflow combines class/interface intersection bounds, inherited and permuted inner classes, constructor-local shadowing, array formals and typed/raw aliases. The previous output failed Go compilation on its bound fields, subtype constraints and invariant storage. The compiler now certifies those declaration-owned layouts and keeps their first-bound storage views while preserving the logical value of an assignment. The unchanged full witness and three authored companions match fresh JDK21 observations; existing refusal, null, identity, method-reference and assignment guards remain intact.

The same integration reuses temporary owner/callable inventories and a pass-local selector table for identifier hygiene. Source-name absence checks remain necessary-only, with complete lexical resolution for possible names, foreign or synthetic graphs, and off-inventory enclosing scopes. Mutations and standalone queries retain fresh fallback behavior. No semantic result cache or source-name case is introduced.

The publication excludes the two private 1220-source acceptance overlay files. They, their original inputs and their results remain preserved outside tracked source. Their absence adds no skip to ordinary regression tests. The additional raw-only owned-syntax optimization is still unrun and is excluded.

Joint normal/race execution and lint are verified before publication. Six original nominal-hygiene subcases run separately at the unchanged native limit; no aggregate timeout is credited as a pass. Full Netty query/strict conversion, original Supplier52/application28, full Gson JVM/Go parity and the round-boundary accumulated suite remain required.
