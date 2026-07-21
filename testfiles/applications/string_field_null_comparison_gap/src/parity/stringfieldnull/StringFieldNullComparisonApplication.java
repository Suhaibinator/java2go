package parity.stringfieldnull;

class StringState {
    String implicit;
    String empty = "";
    String assigned;

    StringState() {
        assigned = "value";
    }
}

public final class StringFieldNullComparisonApplication {
    public static void main(String[] args) {
        StringState value = new StringState();
        System.out.println("implicit=" + (value.implicit == null));
        System.out.println("empty=" + (value.empty == null));
        System.out.println("assigned=" + (value.assigned == null));
    }
}
