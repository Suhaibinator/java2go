# Sol checkpoint 157: Unicode line terminators in parser input

Java processes eligible Unicode escapes before comment boundaries. The parser previously hid valid declarations after an escaped LF or CR in a line comment. ParseAST now supplies a private equal-width lexical view and uses that same view during parenthesized-assignment recovery. Original source bytes and byte-offset maps remain unchanged. This does not implement general Unicode translation.

Independent verification compiled eleven lexical controls with pinned JDK21, reproduced the old parser failure with the same new regression tests, and passed all five parsing parents plus twenty repetitions of the two new parents. All678 runtime and17 symbol parents passed with race detection, as did compiler Unicode diagnostic-mapping controls, default/F1 race builds and test compilation, and golangci-lint2.15 with zero issues. There were31 supervised receipts:30 positive and one expected behavioral regression on the old parser.

Physical diagnostic rows after escaped terminators remain a separate known gap: reportUnsupported still uses parser rows. General escaped-identifier grammar support, the class-facts consumer, complete Throwable integration, full compiler CI and full Netty parity remain open. No generated-program performance gain is claimed. Last accepted full checkpoint remains18; the Sol-only campaign continues.

Verbose evidence stays outside tracked source under root-public157/run/ROOT_RAW_AUDIT.json.
