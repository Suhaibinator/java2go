# Sol checkpoint 153: numeric wrapper width constants

BYTES and SIZE on Byte, Short, Integer, Long, Character, Float and Double now emit Java int constants and supply matching int result metadata. This allows numeric inference, overload selection and compound assignment such as `offset += Long.BYTES`. Boolean gains no invented fields; source declarations, type parameters, imported homonyms and local values retain their existing ownership rules.

The coordinator applied one producer-function delta and one additive regression file to public152. Fresh compiled baseline regressions fail type/emission and compound-assignment assertions, and the original CLI consumer fails strict translation. The candidate passes all7 new and15 adjacent compiler parents, fresh JDK9 and Go6 complete observations across canonical and shadow/donor applications, all672 runtime and17 symbol parents, default/f1 all-package race build/test compilation and golangci-lint2.15 with zero issues. Generated Go leaves and original Java inputs remain unchanged across verification.

This closes the focused Long.BYTES failure observed in ByteBufUtil; full Netty completion is not established. The separate private full1220 normal CLI run still timed out at300 seconds, and its diagnostic profiles remain outside acceptance. Compiler performance guards, source FileChannel construction, Class.cast and Throwable controls are private successors, not imported here. Full CI/round, dependency-source closure, stack/callback contracts and Netty JVM/Go parity remain open. Last accepted full checkpoint remains18; no Java-versus-Go performance advantage is claimed.

Raw verification artifacts remain outside tracked source under `root-public153/run/ROOT_RAW_AUDIT.json`; campaign-state.json records the active resume pointer.
