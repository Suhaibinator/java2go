package parity.workflow.rules;

import parity.workflow.model.WorkflowTask;

public class PayloadPrefixRule implements TaskRule<String> {
    private final String forbiddenPrefix;

    public PayloadPrefixRule(String forbiddenPrefix) {
        this.forbiddenPrefix = forbiddenPrefix;
    }

    public String name() {
        return "payload-prefix";
    }

    public RuleResult evaluate(WorkflowTask<String> task) {
        if (task.getPayload().startsWith(forbiddenPrefix)) {
            return RuleResult.reject("blocked-prefix-" + forbiddenPrefix);
        }
        return RuleResult.allow();
    }
}
