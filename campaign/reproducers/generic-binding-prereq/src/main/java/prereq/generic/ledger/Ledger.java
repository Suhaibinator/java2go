package prereq.generic.ledger;

import org.apache.commons.lang3.mutable.MutableInt;
import org.apache.commons.lang3.mutable.MutableLong;
import prereq.generic.domain.Entry;

public final class Ledger<Balance extends Number> {
    private final Balance starting;
    private final MutableLong total;
    private final MutableInt committed = new MutableInt();
    private final String constructorRoute;
    private final StringBuilder history = new StringBuilder();

    public Ledger(Integer opening, Balance normalized) {
        this((Number) opening, normalized, "I");
    }

    public <Opening extends Number> Ledger(Opening opening, Balance normalized) {
        this((Number) opening, normalized, "G");
    }

    private Ledger(Number opening, Balance normalized, String constructorRoute) {
        this.starting = normalized;
        this.total = new MutableLong(opening.longValue());
        this.constructorRoute = constructorRoute;
    }

    public Balance starting() {
        return starting;
    }

    public int post(int amount, Entry<?> entry) {
        return apply(Integer.valueOf(amount), entry, 1, 'p');
    }

    public int post(Integer amount, Entry<?> entry) {
        return apply(amount, entry, 2, 'b');
    }

    public <Adjustment extends Number> int post(Adjustment amount, Entry<?> entry) {
        return apply(amount, entry, 3, 'g');
    }

    private int apply(Number amount, Entry<?> entry, int fee, char route) {
        String key = entry.touch();
        history.append(route);
        if (amount.longValue() < 0) {
            history.append('X');
            throw new IllegalArgumentException(key);
        }
        total.add(amount.longValue() + fee);
        committed.increment();
        history.append('[').append(key).append(':').append(total.longValue()).append(']');
        return committed.intValue();
    }

    public String snapshot() {
        return constructorRoute + "/" + starting.intValue() + "/" + total.longValue()
                + "/" + committed.intValue() + "/" + history;
    }
}
