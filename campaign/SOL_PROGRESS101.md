# Sol progress101: Map callbacks and strict diagnostics

Map equality and putAll now preserve source-defined Map callbacks, Java exception ordering, caller execution and partial writes. Empty-source TreeMap putAll still performs its required callbacks. ClassCastException messages preserve Java string identity and UTF16 values.

Strict transpilation rejects unsupported assignments and actual invalid Go AST nodes before publishing output. Benign identifiers, warnings, ignored cases and permissive behavior remain covered. An old generated-output comparison failed because checkpoint100 adds a class-literal constant; the original failure is preserved, and fresh untouched-checkpoint100 outputs provide the matching compiler ABI baseline. JVM expectations are unchanged.

Matching verification passed 46 Map controls in both modes, the original 33-command Map application gate, all 1,781 named native race/coverage controls, repository build and vet, uncapped default lint with zero issues, 19 strict controls, four strict rejections and eight supported/permissive CLI comparisons. Source and raw receipts were independently verified before publication.

This is a scoped implementation checkpoint. Existing failures remain visible and unwaived; full CI and full-round acceptance remain open. Round18 is the latest accepted full round. Twenty GPT-6.1 Sol workers continue implementation, challenge generation, performance work and independent review. Original Commons27, Gson118 and Netty applications remain active, alongside a supplemental reflection workflow whose failure is reproducible on the JVM-derived oracle.

Next work repairs source Comparator representations, subtype bounds, generic reflection metadata, reflection argument boxing, generic callable storage and concurrent-map contracts. Netty's corrected private lint gate is green; full translation and application parity are still required. No universal Java, full-Netty or faster-than-Java claim is made.
