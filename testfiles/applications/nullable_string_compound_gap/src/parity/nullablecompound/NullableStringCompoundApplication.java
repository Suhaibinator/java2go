package parity.nullablecompound;

class CompoundState {
    String field;
}

public final class NullableStringCompoundApplication {
    public static void main(String[] args) {
        String local = null;
        try {
            local += "x";
        } catch (NullPointerException failure) {
            local = "local-panic";
        }

        CompoundState state = new CompoundState();
        state.field += "x";
        System.out.println(local + "|" + state.field);
    }
}
