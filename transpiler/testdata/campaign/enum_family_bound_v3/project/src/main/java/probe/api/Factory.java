package probe.api;

public interface Factory {
    <T> Carrier<T> retain(Carrier<T> value);
}
