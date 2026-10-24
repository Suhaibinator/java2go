# Sol progress89: heap buffer compaction

Heap ByteBuffer.compact now copies the remaining window within the shared backing array, sets position to the remaining length, restores limit to capacity and discards the mark. Receiver identity, aliases and byte order are preserved. Fluent lowering supports chained calls.

Frozen native and generated build failures preceded the general implementation. Fresh matching public88 verification passes 13 supervised stages: compile20, 25 exact selected test actions including native race checks, fresh JDK21 execution, strict transpilation, all generated race builds and the race executable. Java and Go produce identical 419-byte observations; both generated Go files remain unchanged after translation. Independent review reports no introduced findings, and the coordinator checks raw streams, cleanup and every source preimage before publication.

This is mutable heap-buffer evidence. Direct/read-only buffers, Java-faster performance, complete Netty translation and full-round acceptance remain open. Netty's current full translation still exceeds 300 seconds. Original Commons27, Gson and accumulated CI must pass before a round is accepted; checkpoint18 remains the latest accepted full round. Continue the Sol-only campaign.
