package parity.localstatic;

public final class LocalStaticMethodApplication {
    static int evaluate() {
        class LocalMath {
            static int offset() {
                return 5;
            }

            static int combine(int value) {
                return value * 2 + offset();
            }

            static int combine(int left, int right) {
                return left + right + offset();
            }
        }

        return LocalMath.combine(6) + LocalMath.combine(-2, 2);
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
