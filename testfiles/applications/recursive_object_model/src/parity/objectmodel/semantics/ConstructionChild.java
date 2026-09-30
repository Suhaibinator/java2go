package parity.objectmodel.semantics;

public final class ConstructionChild extends ConstructionBase {
    private int readyWeight = 7;

    public ConstructionChild() {
        super();
    }

    protected int currentWeight() {
        return readyWeight;
    }

    public int readyWeight() {
        return readyWeight;
    }
}
