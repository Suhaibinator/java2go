# Sol progress102: UUID and Class contracts

Canonical java.util.UUID now supports immutable construction, UTF16 parsing, deterministic name-based hashing, bit access, equality, signed comparison, hashing and string conversion. Canonical Class hashCode and equals use the existing checked object services under the caller execution. Unsupported UUID methods remain strict errors; source declarations and lexical binders retain their own behavior.

Fresh matching verification on the composed checkpoint101 source passed the 42-command UUID gate, 19-command Class/Object gate, all 1,795 named native race/coverage controls, repository build and vet, and uncapped default lint with zero issues. Independent reviews and the coordinator checked source hashes, frozen JVM observations, raw outputs and cleanup before publication. All checkpoint101 Map, strict diagnostics and reflection changes are preserved.

This is a scoped code milestone. Full CI and full dependency application acceptance remain open; round18 remains the latest accepted full round. Commons27, Gson118 and Netty retain their original acceptance applications and unresolved failures.

Twenty GPT-6.1 Sol agents continue separate implementations, adversarial controls, performance measurements and independent verification. Current work covers Comparator identity/default dispatch, generic reflection metadata and varargs, source subtype bounds, generic Map/Collection storage proofs, concurrent-map growth/tree behavior, Object super.finalize and compiler query scaling. A matched JVM/Go performance fixture exposed incorrect interface dispatch before any cost measurements; its oracle stays frozen while a general repair proceeds. No full-Netty or faster-than-Java claim is made.
