package probe.model;

import probe.api.Carrier;

public class EnumCarrier<E extends java.lang.Enum<E>> extends Carrier<E> {
    private E current;
    private int reads;
    private int writes;
    private int casts;
    @Override public E get() { reads++; return current; }
    @Override public void put(E value) { writes++; current = value; remember(value); }
    public int reads() { return reads; }
    public int writes() { return writes; }
    public String state() { return reads + ":" + writes + ":" + snapshots() + ":" + casts; }
    @SuppressWarnings("unchecked")
    public E cast(Object value) { casts++; return (E) value; }
    public <E> E shadow(E value) { return value; }
}
