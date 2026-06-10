public class SwitchExpr {
    static String dayKind(int day) {
        return switch (day) {
            case 1, 2, 3, 4, 5 -> "weekday";
            case 6, 7 -> "weekend";
            default -> "invalid";
        };
    }

    static int score(String grade) {
        int points = switch (grade) {
            case "A" -> 4;
            case "B" -> 3;
            case "C" -> 2;
            default -> {
                yield 0;
            }
        };
        return points;
    }

    public static void main(String[] args) {
        for (int d = 0; d <= 8; d++) {
            System.out.println(d + ": " + dayKind(d));
        }
        System.out.println(score("A"));
        System.out.println(score("B"));
        System.out.println(score("F"));
    }
}
