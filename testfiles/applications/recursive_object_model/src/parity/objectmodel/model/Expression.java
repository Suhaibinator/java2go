package parity.objectmodel.model;

public interface Expression {
    int evaluate(int input);

    String tag();

    default String summary(int input) {
        return tag() + "=" + evaluate(input);
    }
}
