public class DiamondOperator<T> {
    T value;

    public DiamondOperator() {}

    public DiamondOperator(T val) {
        this.value = val;
    }

    // Diamond operator case: new DiamondOperator<>()
    // The type should be inferred from the variable declaration
    public static void testDiamondOperator() {
        DiamondOperator<String> diamond = new DiamondOperator<>();
        DiamondOperator<Integer> diamondWithArg = new DiamondOperator<>(42);
    }

    // Raw type case: new DiamondOperator()
    // This is deprecated but valid Java - no type inference should happen
    @SuppressWarnings("rawtypes")
    public static void testRawType() {
        DiamondOperator raw = new DiamondOperator();
        DiamondOperator rawWithArg = new DiamondOperator("test");
    }

    // Explicit type arguments case: new DiamondOperator<String>()
    public static void testExplicitType() {
        DiamondOperator<String> explicit = new DiamondOperator<String>();
        DiamondOperator<Integer> explicitWithArg = new DiamondOperator<Integer>(42);
    }
}
