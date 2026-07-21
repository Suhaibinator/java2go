package parity.nullablefield;

class TextBase {
    String observed = read();

    TextBase() {}

    String read() {
        return "base";
    }
}

class TextChild extends TextBase {
    String ready = "ready";

    TextChild() {
        super();
    }

    String read() {
        return String.valueOf(ready);
    }
}

public final class ConstructorNullableFieldApplication {
    public static void main(String[] args) {
        TextChild value = new TextChild();
        System.out.println(value.observed + ":" + value.ready);
    }
}
