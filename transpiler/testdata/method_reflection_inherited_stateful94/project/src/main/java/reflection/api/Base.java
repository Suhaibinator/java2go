package reflection.api;
public class Base<T> {
 private T stored;
 public int calls;
 public void set(T value) { stored = value; }
 public T value() { calls++; return stored; }
 public T echo(T value) { calls++; return value; }
}
