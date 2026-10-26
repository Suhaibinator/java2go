# Sol checkpoint 200: measured performance and open application gates

All 45 forks of the prepared fixed-input benchmark completed with identical semantic observations, zero timeouts and all owned process groups absent. The comparison used a checkpoint-191-derived baseline and the isolated null4398 repair, with actual regenerated Go and unchanged dependency implementations. Three seeds, five repeats and all three arms were retained; no observations were excluded.

The repaired Go loop had 8.36% lower median paired elapsed time than baseline Go (15 of 15 pairs, range 2.00–14.49%). Median internal loop times were 6.804 ms for JDK21, 83.234 ms for baseline Go and 76.338 ms for repaired Go. The median paired repaired-Go/JDK elapsed ratio was 11.14: Java was faster in this measured loop. Whole-process median durations were approximately 142/149/148 ms respectively; the loop ratio is not a program startup ratio.

This measures one fixed workload on one host. JIT simplification may reduce actual machine work, declared source calls are not machine-operation counts, and Java thread allocation counters differ from Go process counters. These results establish neither language-wide performance nor full-application speed. The coordinator independently verified all 45 receipts, raw outputs, metrics, pairing and cleanup. Raw observations hash: `61fd2fea169d94ec1c3ea50eb9b408236c9c7f48ddb9242db3d097ef2f69588c`; coordinator audit: `ff3d2fcdc75e0239a1f27fbd20998482dfb7d6a7bed10799642f71220607d31b`.

The original 1,220-source Netty translation with checkpoint 199's qualified normal compiler timed out after 300.032 seconds. Eleven file starts were observed, with ten nonempty generated files; starts do not establish completed translations. This is an open throughput blocker, not a pass or a speed comparison with the earlier instrumented diagnostic. Supplier, Gson and full Netty application acceptance remain open.

This checkpoint changes campaign evidence only. Compiler, runtime, tests and Java fixtures remain byte-exact to checkpoint 199.
