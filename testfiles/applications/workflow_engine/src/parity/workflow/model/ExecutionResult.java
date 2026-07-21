package parity.workflow.model;

public class ExecutionResult {
    private final ResultType type;
    private final int code;
    private final String detail;

    private ExecutionResult(ResultType type, int code, String detail) {
        this.type = type;
        this.code = code;
        this.detail = detail;
    }

    public static ExecutionResult success(int code, String detail) {
        return new ExecutionResult(ResultType.SUCCESS, code, detail);
    }

    public static ExecutionResult retryable(int code, String detail) {
        return new ExecutionResult(ResultType.RETRYABLE_FAILURE, code, detail);
    }

    public static ExecutionResult failure(int code, String detail) {
        return new ExecutionResult(ResultType.PERMANENT_FAILURE, code, detail);
    }

    public ResultType getType() {
        return type;
    }

    public int getCode() {
        return code;
    }

    public String getDetail() {
        return detail;
    }
}
