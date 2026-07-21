package parity.objectmodel.semantics;

public class ConstructionBase {
    private final int observedDuringBaseInitialization = currentWeight();

    public ConstructionBase() {}

    protected int currentWeight() {
        return -1;
    }

    public int observedDuringBaseInitialization() {
        return observedDuringBaseInitialization;
    }
}
