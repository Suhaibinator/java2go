package parity.objectmodel.model;

public final class ConstantExpression implements Expression {
    private final int value;

    public ConstantExpression(int value) {
        this.value = value;
    }

    public int evaluate(int input) {
        return value;
    }

    public String tag() {
        return "const";
    }
}
