# Synthetic owner fixture correction checkpoint 242

The private-helper ownership test copied a generic Box<T> scope onto a real Left<String> anonymous event with zero carried parameters. The strengthened anonymous-layout proof correctly rejected that inconsistent metadata before the test reached its intended source-key check.

Only the synthetic test setup changes: use the actual implemented interface and carried-parameter list, and explicitly reject the original foreign Box<T> metadata. All eight original Java literals and existing negative assertions remain intact; production code is unchanged. The unchanged local full-arity case still tests private-helper declaring-owner refusal, and existing full-arity anonymous guards test declaration/source identity independently.

The exact hosted241 parent/subcase failure was reproduced. The corrected parent/subcase and adjacent guards pass in68 bounded stages:32 real parents and37 children per normal/race mode, with both exact2701-name registries preserved. Source/plan and recorded actual evidence received independent clearance. No tests, assertions, timeouts or strict-mode checks were skipped or weakened.

This is a fixture-fidelity repair. It does not grant new Java compatibility or relabel the historical241 CI failure as passing. Hosted verification on the new head remains required; full Gson112 and Netty acceptance remain open.
