# Milestone 114: canonical syntax and lambda regressions

Six existing regression test leaves now check canonical JavaString and execution callback shapes. Runtime observations evaluate Run once, reject nil, and compare exact UTF16 with the unchanged expected text. Pattern binding, SAM inference, import alias safety, source nil behavior and initialization ordering checks remain. Compiler and runtime code are unchanged.

Sol freshly reproduced seventeen failing parents across the two slices before editing. Independent review verified47 unchanged Java source assignments and preserved observations. The coordinator rebuilt the current race binary and passed 49 focused and adjacent tests in three bounded gates; generated child tests also use race instrumentation.

Other hosted failures remain active and full campaign acceptance remains round18. Raw evidence and hashes are retained under ignored .campaign/resume-20261002.
