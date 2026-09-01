package parity.analytics.pipeline;

public class Rejection {
    private int rowNumber;
    private String code;
    private String detail;

    public Rejection(int rowNumber, String code, String detail) {
        this.rowNumber = rowNumber;
        this.code = code;
        this.detail = detail;
    }

    public int getRowNumber() {
        return this.rowNumber;
    }

    public String getCode() {
        return this.code;
    }

    public String getDetail() {
        return this.detail;
    }
}
