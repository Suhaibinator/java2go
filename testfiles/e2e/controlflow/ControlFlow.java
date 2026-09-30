public class ControlFlow {
    public static void main(String[] args) {
        // if / else if / else
        for (int n = 0; n < 5; n++) {
            if (n == 0) {
                System.out.println("zero");
            } else if (n % 2 == 0) {
                System.out.println("even " + n);
            } else {
                System.out.println("odd " + n);
            }
        }

        // while
        int i = 3;
        while (i > 0) {
            System.out.println("while " + i);
            i--;
        }

        // do-while
        int j = 0;
        do {
            System.out.println("do " + j);
            j++;
        } while (j < 2);

        // nested loop with break / continue
        int sum = 0;
        for (int a = 0; a < 4; a++) {
            for (int b = 0; b < 4; b++) {
                if (b == 2) {
                    continue;
                }
                if (a + b > 4) {
                    break;
                }
                sum += a * b;
            }
        }
        System.out.println("sum " + sum);

        // classic switch statement
        for (int k = 0; k < 4; k++) {
            switch (k) {
                case 0:
                    System.out.println("k0");
                    break;
                case 1:
                case 2:
                    System.out.println("k12");
                    break;
                default:
                    System.out.println("kdef");
            }
        }

        // ternary
        int max = (7 > 3) ? 7 : 3;
        System.out.println("max " + max);
    }
}
