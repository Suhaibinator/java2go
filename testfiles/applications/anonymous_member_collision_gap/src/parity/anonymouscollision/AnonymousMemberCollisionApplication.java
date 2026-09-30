package parity.anonymouscollision;

public final class AnonymousMemberCollisionApplication {
    static int evaluate() {
        var value = new Object() {
            int score = 4;

            int score() {
                return score;
            }
        };

        return value.score * 10 + value.score();
    }

    public static void main(String[] args) {
        System.out.println(evaluate());
    }
}
