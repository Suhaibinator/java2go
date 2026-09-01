package parity.objectmodel.semantics;

public final class EvenState {
    public boolean accepts(int value, OddState odd) {
        return value == 0 || odd.accepts(value - 1, this);
    }
}
