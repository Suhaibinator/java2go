---
name: shared-worktree-not-isolated
description: java2go-roadmap team agents share one working dir, not isolated worktrees; build flaps red from concurrent WIP
metadata:
  type: project
---

On the java2go-roadmap team, the task briefs claim each agent works in "an isolated git worktree," but in practice all agents edit the SAME working directory (/Users/suhaib/code/java2go, branch newBranch) concurrently. Hot files like transpiler/expression.go, statement.go, and declaration.go are co-edited by multiple tasks at once.

**Why:** Consequence — `go build ./...` and `go test ./...` flap red continuously because teammates push incomplete code referencing not-yet-defined symbols (observed: collectionNeedsSliceForRange, synchronizedMethodPrologue, objectCreationClassBody, lowerAnonymousClass, isExceptionJavaType, tryInstanceIntrinsic). These reds are almost never your own code.

**How to apply:** To verify your own changes, poll for a green build window with an until-loop (`for i in $(seq 1 30); do go test ./... && break; sleep 12; done`) and run your specific tests then. Don't conclude your change regressed something until you've seen the failure in a stable green window — re-run the specific test a few times. Do NOT `git add` co-edited files (expression.go/statement.go/declaration.go): it sweeps teammates' WIP into your commit. Ask team-lead how commits are coordinated before committing; safe-to-commit-alone files are ones only you touched. See [[task1-diagnostics-system]].
