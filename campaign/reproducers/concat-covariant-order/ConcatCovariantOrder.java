class CounterBase {
    int count;
    public CounterBase touch(Object ignored) { count++; return this; }
}
class CounterChild extends CounterBase {
    public CounterChild touch(Object ignored) { count += 10; return this; }
}
public class ConcatCovariantOrder {
    public static void main(String[] args) {
        CounterChild counter = new CounterChild();
        CounterBase view = counter;
        System.out.println(counter.count + ":" + (view.touch("x") == counter));
    }
}
