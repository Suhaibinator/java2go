# Sol checkpoint 196: plain member-type parsing

The member-type parser returns already trimmed names directly when they contain no generic delimiters. Names containing either delimiter retain the existing parser and diagnostics. This removes two or three allocations per query in the six measured plain-name cases; it does not establish a whole-program speed gain.

The regression was observed before repair. Both new parents passed twenty normal and twenty race repetitions, covering 32 semantic and warning cases, with seven existing shields in each mode. Full race suites passed all 849 runtime, 23 symbol and 10 parser parents. Default and F1 whole-project race builds, test compilation and lint passed. Compiler native checks were repeated from the package directory. Two overlapping orchestration receipts remain excluded; fresh serial successors passed.

Hosted checkpoint 195 reduced compiler failures from 45 to 30, with all 15 addressed parents passing and no new failure names. The remaining failures and original Supplier28, Gson112 and Netty1220 application gates remain open.
