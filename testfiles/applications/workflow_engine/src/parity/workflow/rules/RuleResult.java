package parity.workflow.rules;

public class RuleResult {
    private final boolean allowed;
    private final String reason;

    private RuleResult(boolean allowed, String reason) {
        this.allowed = allowed;
        this.reason = reason;
    }

    public static RuleResult allow() {
        return new RuleResult(true, "ok");
    }

    public static RuleResult reject(String reason) {
        return new RuleResult(false, reason);
    }

    public boolean isAllowed() {
        return allowed;
    }

    public String getReason() {
        return reason;
    }
}
