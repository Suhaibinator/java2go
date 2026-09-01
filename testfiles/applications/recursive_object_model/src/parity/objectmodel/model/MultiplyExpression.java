package parity.objectmodel.model;

public final class MultiplyExpression implements Expression {
    private final Expression left;
    private final Expression right;

    public MultiplyExpression(Expression left, Expression right) {
        this.left = left;
        this.right = right;
    }

    public int evaluate(int input) {
        return left.evaluate(input) * right.evaluate(input);
    }

    public String tag() {
        return "mul";
    }
}
