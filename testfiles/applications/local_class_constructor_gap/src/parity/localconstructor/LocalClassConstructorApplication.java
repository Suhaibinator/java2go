package parity.localconstructor;

public final class LocalClassConstructorApplication {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    static String evaluate() {
        effects = 0;

        class LocalBox {
            int value;

            LocalBox(int left, int right) {
                effects = effects * 10 + 3;
                value = left * 10 + right;
            }

            int read() {
                return value;
            }
        }

        LocalBox box = new LocalBox(mark(1, 4), mark(2, 5));
        return box.read() + ":" + effects;
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
