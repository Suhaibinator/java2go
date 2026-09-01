package parity.workflow.model;

public enum Priority {
    LOW(10),
    NORMAL(20),
    HIGH(30),
    CRITICAL(40);

    private final int weight;

    Priority(int weight) {
        this.weight = weight;
    }

    public int getWeight() {
        return weight;
    }
}
