package parity.localfieldinit;

public final class LocalClassFieldInitializerApplication {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    static String evaluate() {
        effects = 0;
        mark(1, 0);

        class LocalValue {
            int first = mark(2, 7);
            int second = mark(3, first + 1);

            int sum() {
                return first + second;
            }
        }

        LocalValue value = new LocalValue();
        mark(4, 0);
        return value.first + ":" + value.second + ":" + value.sum() + ":" + effects;
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
