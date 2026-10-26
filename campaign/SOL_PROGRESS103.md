# Milestone 103: canonical String observations in CI tests

The Ubuntu Unit job on revision `65ed9734` reported 18 class-initialization failures because two generated **test** templates compared canonical JavaString pointers with Go string literals. The templates now call `Run()` once, fail explicitly on null, and decode UTF-16 content for the original string comparison.

All 21 embedded Java source blocks, 18 expected strings, test names and comparison assertions are preserved. The compiler and runtime are unchanged.

TDD reproduced two representative compilation failures before the fix. A fresh race-enabled native test build then ran all 18 original tests: 18 passed, zero failures or skips, with clean process cleanup. Independent source, reverse-patch and raw-execution reviews passed. The coordinator separately rehashed all 3,110 source members and all four captured streams and checked the exact 18 test names.

- Root review: `/private/tmp/java2go-root-classinit103-review.json`, SHA-256 `f0b3445a4575d49d12c04c53907bb1148edc9329bb9a201c4229e0bc337cea9a`.
- Independent execution review: `/private/tmp/java2go-ci-classinit-observation103-peer-review/ACTUAL-GREEN18-REVIEW.json`.
- Evidence: `/private/tmp/java2go-ci-classinit-observation103-green-v1`.

Full CI remains unaccepted. The Sol teams continue the file/thread String boundaries, collector callback, String.contains, IntSummaryStatistics and reflection repairs. Existing generic compiler and dependency application failures remain visible. Full campaign round18 remains the latest accepted round.
