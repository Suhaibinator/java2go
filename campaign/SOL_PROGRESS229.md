# Sol checkpoint 229: collection-call regression contracts

Two CI guards still expected a direct native Add selector after source collection lowering moved to CollectionAddExecution. The updated tests validate the canonical helper, exact execution context, receiver evaluation, complete parser result flow, null handling and operation order. Twelve existing negative controls remain; fourteen new controls reject corrupt helpers, duplicated evaluation and parser calls, wrong returns, and unreachable correct-looking calls.

Both focused parents and all 26 negative controls pass in normal and race runs. The unchanged five-source field workflow also passes its actual JVM/Go oracle comparison and generated Go race build. Independent actual review: `700fbdffd194743f71cdb690f0637eab4bdc71cbac36c6dfc3a47bbb8d9af93b`. No production code or Java fixture changed.

Published-head CI remains pending. The initial failures and rejected validator draft remain recorded; full Gson and Netty application acceptance is still open.
