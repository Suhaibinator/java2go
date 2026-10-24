# Sol-only campaign allocation

The current campaign uses GPT-6.1 Sol agents with high reasoning effort exclusively. Sol owns Java challenge authoring, independent JVM oracles, all compiler/runtime implementation, the hardest semantics, build integration, independent verification and dedicated performance research. No Astra agents may be spawned, resumed or assigned work. Existing Astra results remain immutable historical evidence.

The goal is successive independent Java challenge, general transpiler repair and differential verification rounds toward complex dependency-based applications and Netty. Correctness takes priority over performance. Translate real dependency implementations; JVM delegation, handwritten dependency replacements and edited generated Go do not count. Passing the accumulated corpus demonstrates tested compatibility rather than support for every Java application.

Challenge authors own Java inputs and JVM-derived expectations; implementation owners cannot weaken frozen challenges. The coordinator assigns up to three heavy correctness lanes with separate processes, private outputs and GOMAXPROCS=2. Performance measurement runs exclusively. Preserve existing bounds, rejected oracles and pre-existing failures. Root alone integrates managed files and periodically commits and pushes verified milestones to origin branches.

Sol hard-semantics owner: sol_hard_semantics_v10. Dedicated performance owner: sol_performance_v10, with sol_bigmath_audit_v6 preparing executable measurement harnesses. Other Sol owners and exact resume points are recorded in campaign-state.json and SOL_PROGRESS58.md. Full Gson remains required and unaccepted; Netty progression stays gated.
