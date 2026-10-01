# Sol progress84: canonical String chars streams

JavaString chars streams now copy exact UTF16 code units, including NUL and paired or unpaired surrogates. Null receivers fail when chars is called, and the stream cannot alias String storage. Original Java oracle source and complete observations are unchanged. Test drivers and three structural assertions explicitly migrate to the canonical JavaString representation; obsolete assertions and missing-helper RED receipts remain preserved.

Matching current83 validation passes both20-package compilation checks, native2 race checks and all33 selected compiler checks (original30 plus3 behavior regressions). All four historical failures pass. Generated behavior modules run normally against actual pinned JDK21 output; the outer compiler binary runs with race instrumentation. The coordinator independently reread all13 terminal supervision receipts, durable channels and exact test inventories, checked the five publication files, and protected Atomic6/NIO10 hashes.

This is a scoped component milestone. Full Java Stream laziness and close behavior, accumulated current CI/round acceptance, translated Netty and performance superiority remain unaccepted. Checkpoint18 is still the latest accepted full round. No timeout, compilation failure or unsupported behavior counts as parity.

Resume the Sol-only parallel repair campaign: comment-aware AST traversal, volatile/reflection and functional updater behavior, Map/Future timeout diagnosis, Gson generic object lowering, Maven build export and unchanged real Netty ingestion. Preserve original challenges and user untracked performance files. Root remains the sole public and Git writer.
