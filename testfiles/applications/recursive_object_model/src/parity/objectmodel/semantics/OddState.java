package parity.objectmodel.semantics;

public final class OddState {
    public boolean accepts(int value, EvenState even) {
        return value != 0 && even.accepts(value - 1, this);
    }
}
