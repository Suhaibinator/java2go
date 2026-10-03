package reflection.api;
public class Base<T> implements ValueContract<T> {
    public T value() { return null; }
    public T echo(T value) { return value; }
    public String unchanged() { return "base"; }
    public int count() { return 3; }
    protected String hidden() { return "hidden"; }
}
