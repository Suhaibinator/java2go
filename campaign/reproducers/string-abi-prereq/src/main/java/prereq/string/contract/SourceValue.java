package prereq.string.contract;

public final class SourceValue {
    private final String stored;

    public SourceValue(String stored) {
        this.stored = stored;
    }

    @Override
    public String toString() {
        Ledger.stringifications++;
        Ledger.steps.append('T');
        return stored;
    }
}
