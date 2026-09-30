# Native oracle UTF-8 checkpoint64

The Java oracle helper now explicitly requests UTF-8 stdout and stderr under a C locale. JDK21 otherwise emits ASCII replacement bytes despite file.encoding and the default Charset being UTF-8. Production compiler/runtime behavior, Java fixtures, expected observations, Locale, native encoding and file encoding are unchanged.

The same new stdout/stderr byte regression is RED before the two flags and GREEN afterward. Independent actual audit: 58430781cabdc1cc0cabccf5b6a6a3c7302ad4843e67a43204b1b768144a0d24. Patch: 3ebc2f0bb8aa30944687fccb5a7c9c2a3599319bac4fb614059b9bd716e8fde5. All 1984 files match frozen source manifest809849bfe618daf0738f6aad065419a40ff8295d79bcc1223464f735d7b484ca, apart from additional publication notes.

Prior source1983 passes the independently verified native all16 gate: 119 stages, 128 raw byte comparisons, 377 stdjava parents and 219 subtests. This new test-launch change requires fresh affected gates. The ten-parent gate preserves all nine historical failed assertions and adds the two-stream regression; it is separately supervised and tracked. Full CI is not accepted by this checkpoint, and the nine failures are not waived. Verbose artifacts remain outside tracked source.

Continue the active Sol-only campaign with the ten-parent actual gate and remaining CI shards. No full coherent, Codec, Gson, Netty or performance acceptance follows from this checkpoint.
