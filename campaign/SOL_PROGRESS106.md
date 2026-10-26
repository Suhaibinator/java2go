# Milestone 106: canonical String observations in remaining CI scaffolds

Four existing generated test scaffolds now decode canonical JavaString UTF16 content after checking null and evaluating each tested method once. The enhanced-for source checks now name the canonical JavaString component type. All embedded Java, original expected observations, semantic fragment checks and failure assertions are preserved.

The current103 Ubuntu Unit log retained the original compilation and stale-fragment failures. Fresh native test builds and the seven original affected roots passed in both normal and race-enabled runs: 14 passing observations, zero failures or skips, clean process cleanup. Independent source and execution review passed; the coordinator separately replayed all 12 guarded edits, rehashed 3,111 source members and verified the raw command evidence.

The four disjoint test leaves were transported onto105, preserving its reflection getter and104 portable JDK helper. The compiler and runtime are unchanged by this milestone. Full CI remains unaccepted: larger compiler and dependency application failures remain active, and round18 is the latest full campaign acceptance.
