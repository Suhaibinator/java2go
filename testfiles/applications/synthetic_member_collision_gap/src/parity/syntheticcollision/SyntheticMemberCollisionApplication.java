package parity.syntheticcollision;

public final class SyntheticMemberCollisionApplication {
    static int evaluate() {
        class LocalValue {
            int value;

            int value() {
                return value;
            }
        }

        LocalValue local = new LocalValue();
        local.value = 7;
        return local.value * 10 + local.value();
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
