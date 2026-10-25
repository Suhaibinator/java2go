# Dependency-facing JDK services checkpoint 235

This checkpoint adds the exercised URL/URI, numeric and loopback address, date formatting, and deque contracts needed by the dependency campaign. Java dependency implementation sources remain translated; the new native implementations provide JDK/platform services. The shared Date bridge change retains declaration ownership and the protected cross-package restriction from checkpoint 234.

All 317 fresh qualification stages passed and were independently audited. Compiler tests covered 186 parents in each of normal and race modes; the full runtime covered 939 parents per mode. Both original 4059 budget parents passed with real atomic coverage and race instrumentation, and golangci-lint 2.15 reported no issues. Supplemental deque preservation tests passed in both modes.

Six fresh Java projects covered four URL workflows, a date reducer, and a deque workflow. Fourteen fresh JVM executions and 28 Go executions matched complete declared output streams; strict conversion had zero warnings for all six projects, and generated packages and entry programs built in normal and race modes. Original inputs, JVM-derived expectations, and generated Go implementation bytes were preserved. Three harness-only validator/partition corrections retain their original successful raw results and disjoint original test cases.

The baseline public 234 CI run passed all 2,600 compiler parents without failures or skips. Its remaining application failure was the unsupported URL call in Gson's TypeAdapters.java. New-head CI must verify the next failure stage; this checkpoint does not claim full Gson or Netty compatibility or a completed campaign round. Broader DNS/provider/caching behavior and unexercised JDK configurations remain unverified.

The campaign remains restricted to Sol agents. Ongoing cleanup may remove only closed, checkpointed generated executables, retaining sources, inputs, logs, tools, active baselines, and comparison executables.
