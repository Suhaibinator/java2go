package parity.localrecursion;

public final class LocalClassRecursionApplication {
    static int recurse(final int start) {
        class LocalCounter {
            int down(int depth) {
                return depth == 0 ? start : 1 + down(depth - 1);
            }
        }
        return new LocalCounter().down(4);
    }

    public static void main(String[] args) {
        System.out.println(recurse(7));
    }
}
