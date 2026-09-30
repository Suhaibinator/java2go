package parity.workflow.rules;

import parity.workflow.model.WorkflowTask;

public interface TaskRule<T> {
    String name();

    RuleResult evaluate(WorkflowTask<T> task);
}
