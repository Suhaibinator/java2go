# Expanded Sol campaign checkpoint

Twelve GPT-6.1-Sol agents now own routine authoring, TDD repair, harness work and independent verification. Astra is retained for the hardest architecture and optimization design. Four added owners handle Strings, generic runtime representations, Map.Entry and abstract collection defaults. All use high reasoning effort. Existing challenge/oracle ownership remains independent from implementation.

Three correctness lanes are coordinator assigned on the checked 16-core,128-GiB host. Each uses GOMAXPROCS=2, private source/output paths and supervised process trees; existing timeouts are unchanged. Performance measurements run exclusively.

The native full22 run passed its first11 stages, including build, lint and1930 unit checks. Strict parity then exposed a recursive_object_model transpilation panic (slice bounds [:528] with capacity512). The run terminated normally with exit1:53 parity cases passed and this one failed;10 later stages were not run. This gate remains unaccepted; a Sol compiler owner is investigating from a separate copy. Full Gson remains a required failing challenge and Netty progression stays gated.

The ordinary selector repair passed six focused controls and independent audit. Whitespace baseline produced three valid JVM oracles followed by missing-helper Go build failures; its frozen candidate passed nine runtime controls and three generated parity cases, pending independent audit. Duration/Instant encountered a default-package harness problem, explicitly rejected as a semantic red; no repair is justified from that result. Scalar Class representation unit tests recorded the expected red; a narrow repair is in progress.

The literal semantic and Go allocation baseline is independently accepted. The preserved archive has2024 hash-verified files plus its manifest, including frozen implementation, inputs, raw results and independent audits. It records18 allocation rows across three fresh processes. No optimization or Java performance advantage is claimed.

Archive: `.campaign/checkpoints/checkpoint19/literal-v2-accepted/literal-v2-accepted-semantic-allocation.tar.gz`

SHA256: `901b1efae2ef6aa71d24273ef24723e5deff21f479bf80d6eb0f8ff8f2abdebe`

Manifest SHA256: `f37ed381a4b1c94733278aef8dead911692e73b4cedc5944c6f03c31114cb146`

Production compatibility acceptance remains checkpoint18. All later private candidates and partial gates retain their separate status.
