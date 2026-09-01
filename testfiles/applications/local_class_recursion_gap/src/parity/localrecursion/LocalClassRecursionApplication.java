package parity.localrecursion;

public final class LocalClassRecursionApplication {
    static int effects;

    static int mark() {
        effects = effects * 10 + 2;
        return 7;
    }

    static int recurse(final int start) {
        class LocalCounter {
            int down(int depth) {
                return depth == 0 ? start : 1 + down(depth - 1);
            }
        }
        return new LocalCounter().down(4);
    }

    static String nullReceiverOrder() {
        class LocalValue {
            String ping(int ignored) {
                return "body";
            }
        }
        LocalValue value = new LocalValue();
        value = null;
        effects = 0;
        try {
            return value.ping(mark()) + ":" + effects;
        } catch (NullPointerException expected) {
            return "npe:" + effects;
        }
    }

    public static void main(String[] args) {
        System.out.println(recurse(7));
        System.out.println(nullReceiverOrder());
    }
}
