package parity.anonymousrecursion;

abstract class RecursiveAction {
    abstract int apply(int value);
}

public final class AnonymousClassRecursionApplication {
    static int effects;

    static int mark() {
        effects = effects * 10 + 2;
        return 7;
    }

    static int factorial(int value) {
        RecursiveAction action = new RecursiveAction() {
            int apply(int current) {
                return current == 0 ? 1 : current * apply(current - 1);
            }
        };
        return action.apply(value);
    }

    static String nullReceiverOrder() {
        var value = new Object() {
            String ping(int ignored) {
                return "body";
            }
        };
        value = null;
        effects = 0;
        try {
            return value.ping(mark()) + ":" + effects;
        } catch (NullPointerException expected) {
            return "npe:" + effects;
        }
    }

    public static void main(String[] args) {
        System.out.println(factorial(6));
        System.out.println(nullReceiverOrder());
    }
}
