package parity.syntheticcollision;

public final class SyntheticMemberCollisionApplication {
    static int evaluate() {
        class LocalValue {
            int value = 7;

            int value() {
                return value;
            }
        }

        LocalValue local = new LocalValue();
        return local.value();
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
