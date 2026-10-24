# Sol checkpoint 154: declaration-name allocation

The compiler now walks qualified source declaration names with `strings.Cut` instead of allocating a slice with `strings.Split`. It retains the original first matching owner, ordered member lookup, caller context, empty segments, accessibility and source-graph identity. This changes compiler allocation only; it does not establish a generated Go application performance advantage.

The coordinator rebased one producer function and two new helper/test files onto actual checkpoint153. The compiled baseline passes semantic controls but fails all eight zero-allocation cases. The candidate passes those eight cases three times with zero allocations per query, all21 ownership/header/local/anonymous controls, three existing complete JDK/Go projects, all672 runtime and17 symbol parents, and default/f1 all-package race build, test compilation and golangci-lint2.15 with zero issues. Original inputs and generated Go remain unchanged during verification.

Full CI, the accumulated adversarial round and Netty JVM/Go parity remain open. A separate private full Netty compiler run reaches the missing FileChannel write operation in CompositeByteBuf; that source contains additional unpublished prerequisites and is not checkpoint154 application acceptance. Source channel writes, exception stacks/callbacks and coherent integration continue in private test lanes. Last accepted full checkpoint remains18.

Raw verification artifacts remain outside tracked source under `root-public154/run/ROOT_RAW_AUDIT.json`. The campaign remains active with GPT-6.1 Sol agents only.
