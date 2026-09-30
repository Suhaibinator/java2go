package parity.objectmodel.model;

public final class InputExpression implements Expression {
    public int evaluate(int input) {
        return input;
    }

    public String tag() {
        return "input";
    }
}
