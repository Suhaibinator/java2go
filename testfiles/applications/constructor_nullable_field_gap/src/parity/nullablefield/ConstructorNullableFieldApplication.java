package parity.nullablefield;

class TextReader {
    static String inspect(TextChild value) {
        return String.valueOf(value.ready) + "/" + String.valueOf(value.raw());
    }
}

class TextBase {
    String observedFromInitializer = read("base-initializer");
    String observedFromConstructor;

    TextBase() {
        observedFromConstructor = read("base-constructor");
    }

    String read(String phase) {
        return phase + "=base";
    }
}

class TextChild extends TextBase {
    String ready = initializeReady();
    String observedFromChildInitializer = read("child-initializer");
    String observedFromChildConstructor;

    TextChild() {
        super();
        observedFromChildConstructor = read("child-constructor");
    }

    String initializeReady() {
        return "ready";
    }

    String raw() {
        return ready;
    }

    String read(String phase) {
        return phase + "=" + TextReader.inspect(this);
    }
}

public final class ConstructorNullableFieldApplication {
    public static void main(String[] args) {
        TextChild value = new TextChild();
        System.out.println(value.observedFromInitializer);
        System.out.println(value.observedFromConstructor);
        System.out.println(value.observedFromChildInitializer);
        System.out.println(value.observedFromChildConstructor);
        System.out.println(TextReader.inspect(value));
    }
}
