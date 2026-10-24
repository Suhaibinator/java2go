# Sol checkpoint 155: invalid array method expectations

The Thread service negative suite now expects the expression rejection introduced by checkpoint152 for three invalid array method calls. All20 Java fixtures remain byte-identical, and the17 other rows keep their prior behavior. Exact diagnostic count/content, typed strict-mode failure, permissive panic text and native-emitter/result-metadata shields remain required. Production compiler/runtime code is unchanged.

The coordinator reproduced six failing array/mode cases on published154 and independently confirmed that JDK21 rejects all three Java snippets. The updated suite passes all40 row/mode cases, four companion service contracts, ten array/thread shields, four valid array/Object/clone controls, and default/f1 all-package race build, test compilation and golangci-lint2.15 with zero issues. The initial proposal's incorrect permissive-placeholder expectation remains recorded separately; the final expectation uses the pinned shared expression fallback format.

This corrects one CI regression test. Other compiler/dependency CI failures, full accumulated round acceptance and full Netty JVM/Go parity remain open. The active campaign continues with Sol agents, genuine FileChannel writes/resources, exception-stack support and independent integration verification. Last accepted full checkpoint remains18.

Raw evidence remains outside tracked source under `root-public155/run/ROOT_RAW_AUDIT.json`.
