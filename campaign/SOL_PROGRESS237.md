# Covered test framing checkpoint 237

Checkpoint236 CI executed all 17 new filesystem controls on Linux. One test rejected its child test framework's coverage trailer even though the child exited successfully, stderr was empty and the API observations matched the JDK fixture.

The fixture now accepts exactly the plain PASS trailer and the observed PASS plus no-statements coverage trailer. Every API byte, exit status, context and stderr check remains enforced. The runtime, Java inputs and expected observations are unchanged.

Actual covered normal and race runs reproduced the failure before the repair and passed afterward, including all 17 original controls and 11 decoder vectors. An independent audit verified the source, results and recorded cleanup. The initial manual coverage-registry setup attempt remains separate infrastructure evidence. New-head hosted CI remains pending; this test-only repair does not accept a campaign round.
