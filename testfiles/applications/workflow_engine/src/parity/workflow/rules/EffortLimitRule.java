package parity.workflow.rules;

import parity.workflow.model.WorkflowTask;

public class EffortLimitRule<T> implements TaskRule<T> {
    private final int maximum;

    public EffortLimitRule(int maximum) {
        this.maximum = maximum;
    }

    public String name() {
        return "effort-limit";
    }

    public RuleResult evaluate(WorkflowTask<T> task) {
        if (task.getEffort() > maximum) {
            return RuleResult.reject("effort-" + task.getEffort() + "-exceeds-" + maximum);
        }
        return RuleResult.allow();
    }
}
