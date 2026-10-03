# Milestone 122: reflection boundaries and package lookup

Captured local constructor metadata now follows JDK descriptor order and projects validated reflection arguments into the generated capture-first ABI. Constructor callbacks are retained for captured constructors; unary no-argument construction is exposed only for actual reflected zero arity. Java dollar identifiers preserve reflection names while field/accessor lowering uses the allocated Go identity.

Package ownership facts are indexed for the duration of one conversion and restored afterward. Unknown or late declarations retain fresh resolution. This improves repeated ownership lookup without claiming concurrent compiler Run safety or application speedup over Java.

The coordinator rebuilt a matching race binary and replayed all53 selected parents with55-second native deadlines and65-second supervisors. Constructor verification covers15 original/new parents; clone-dependent tests remain a separate gate. Dollar verification covers13 focused/adjacent parents; ownership covers7 and syntax families18. Original Java inputs and observations remain unchanged. Nested race builds are explicit; original helper module modes are preserved.

Public121 CI passes unit/build/application parity/nonparity/fuzz/vulnerability; compiler shards, dependency and lint remain failing. A fresh candidate lint reproduces five monitor-test warnings, assigned to a Sol agent. It is retained as a failure, not a gate pass. Full Gson, Netty and round acceptance remain open. A full1220-source Netty stage clears the allocator finalize blocker but times out at300.25seconds; that timeout is not success. Latest accepted full round remains18.

Exact source hashes, runner receipts, pending reproducers and source audits remain in ignored .campaign/resume-20261002. Continue Sol-only repairs and independent verification.
