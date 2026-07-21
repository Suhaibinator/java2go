package parity.anonymousrecursion;

abstract class RecursiveAction {
    abstract int apply(int value);
}

public final class AnonymousClassRecursionApplication {
    static int factorial(int value) {
        RecursiveAction action = new RecursiveAction() {
            int apply(int current) {
                return current == 0 ? 1 : current * apply(current - 1);
            }
        };
        return action.apply(value);
    }

    public static void main(String[] args) {
        System.out.println(factorial(6));
    }
}
