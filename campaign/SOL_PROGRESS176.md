# Sol checkpoint 176: align Reader regression assertions with the execution ABI

Three compiler test parents still expected direct BufferedReader.ReadLineReference/Close calls after the runtime and canonical Reader contract tests had moved to execution-aware dispatch. Four exact expected-call assertions in two test files now require BufferedStringReaderReadLineExecution or BufferedStringReaderCloseExecution with the current Execution and the original receiver.

All Java fixtures and every other assertion remain byte-exact. The nullable assignment bridge, null comparison, checked String length, absent assignment placeholder, constructors, resource ownership and close checks are retained. Production code and application oracles do not change.

The implementer retained fresh baseline failures for the three parents, then passed all12 parents in the two files, seven canonical Reader contract parents and20 Reader/resource/TWR runtime controls with race checks. Default/F1 package builds and test compilation passed. The coordinator independently joined only the four exact replacements onto actual public175, compiled a new race binary, verified the actual19-parent registry and ran all19 successfully with no skips. All13 implementer and three coordinator raw process receipts and stream hashes were audited, with successful cleanup. Evidence: root-reader176/COORDINATOR-AUDIT.json.

This fixes these reproduced stale assertion failures. Full CI, dependency application parity, the separate unpublished metadata/lambda repair and full Netty remain unaccepted.
