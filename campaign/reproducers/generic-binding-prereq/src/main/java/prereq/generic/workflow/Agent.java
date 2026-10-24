package prereq.generic.workflow;

import prereq.generic.domain.Entry;
import prereq.generic.ledger.Ledger;

public final class Agent<Balance extends CharSequence> {
    private final Balance name;
    private final StringBuilder evaluations = new StringBuilder();

    public Agent(Balance name) {
        this.name = name;
    }

    public <Seed extends Number> Ledger<Integer> openReserve(Seed opening) {
        evaluations.append('o');
        return new Ledger<Integer>(opening, Integer.valueOf(opening.intValue()));
    }

    public int primitive(int value) {
        evaluations.append('p');
        return value;
    }

    public Integer boxed(int value) {
        evaluations.append('b');
        return Integer.valueOf(value);
    }

    public Long wide(long value) {
        evaluations.append('w');
        return Long.valueOf(value);
    }

    public Entry<Balance> entry(String kind) {
        evaluations.append('e');
        return new Entry<>(name, kind);
    }

    public <Dispatch extends Number> int forward(
            Ledger<Integer> ledger, Dispatch amount, Entry<Balance> entry) {
        evaluations.append('f');
        return ledger.post(amount, entry);
    }

    public int boundCheck(Ledger<Integer> ledger) {
        return ledger.starting().intValue() + name.length();
    }

    public String evaluations() {
        return evaluations.toString();
    }
}
