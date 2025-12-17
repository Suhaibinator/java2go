// Test file for advanced enum features: fields, constructor arguments, and methods
enum Status {
    PENDING("pending", 0),
    APPROVED("approved", 1),
    REJECTED("rejected", 2);

    private String value;
    private int code;

    Status(String value, int code) {
        this.value = value;
        this.code = code;
    }

    public String getValue() {
        return this.value;
    }

    public int getCode() {
        return this.code;
    }

    public static void main(String[] args) {
        // Test values()
        for (Status s : Status.values()) {
            System.out.println(s.name() + ": " + s.getValue());
        }

        // Test ordinal()
        System.out.println(Status.PENDING.ordinal());

        // Test valueOf()
        Status s = Status.valueOf("APPROVED");
        System.out.println(s.getValue());
    }
}
