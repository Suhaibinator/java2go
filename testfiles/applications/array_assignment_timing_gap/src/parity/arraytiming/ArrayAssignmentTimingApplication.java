package parity.arraytiming;

public final class ArrayAssignmentTimingApplication {
    private static String trace = "";

    private ArrayAssignmentTimingApplication() {
    }

    private static int index() {
        trace = trace + "i";
        return 2;
    }

    private static int rightHandSide() {
        trace = trace + "r";
        return 7;
    }

    public static void main(String[] args) {
        int[] missing = null;
        try {
            missing[index()] = rightHandSide();
        } catch (NullPointerException expected) {
            trace = trace + "c";
        }
        System.out.println(trace);
    }
}
