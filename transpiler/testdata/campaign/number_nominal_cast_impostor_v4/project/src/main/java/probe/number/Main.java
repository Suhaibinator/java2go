package probe.number;

import probe.api.Carrier;
import probe.api.Factory;
import probe.model.NamedFactory;
import probe.model.TextCarrier;
import probe.model.IntegerCarrier;
import probe.model.NumberCarrier;

public final class Main {
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Integer original = Integer.valueOf(seed);
        IntegerCarrier concrete = new IntegerCarrier();
        NumberCarrier<Integer> bounded = concrete;
        Carrier<Integer> parent = concrete;
        Carrier raw = concrete;
        bounded.put(original);
        System.out.println("identity=" + (parent.get() == original) + ":"
                + (bounded.snapshot() == original) + ":" + (raw == parent)
                + ",state=" + bounded.state());
        String marker = "binder-" + seed;
        System.out.println("method-shadow=" + (bounded.shadow(marker) == marker));
        NamedFactory namedFactory = new NamedFactory();
        Factory factory = namedFactory;
        Carrier<Integer> retained = factory.retain(parent);
        TextCarrier text = new TextCarrier();
        String factoryMarker = "factory-" + seed;
        text.put(factoryMarker);
        Carrier<String> retainedText = factory.retain(text);
        Carrier<Integer> absent = factory.retain((Carrier<Integer>) null);
        Carrier<Integer> direct = namedFactory.retain(parent);
        System.out.println("factory=" + (retained == parent) + ":" + (retainedText == text)
                + ":" + (retainedText.get() == factoryMarker) + ":" + (absent == null)
                + ":" + (direct == parent) + ",calls=" + namedFactory.calls());

        try { raw.put("not-number-" + seed); }
        catch (ClassCastException expected) { System.out.println("reject-string=" + bounded.state()); }
        try { raw.put(new probe.shadow.Number()); }
        catch (ClassCastException expected) { System.out.println("reject-shadow=" + bounded.state()); }
        System.out.println("retained=" + (bounded.get() == original) + ",state=" + bounded.state());

        Double pollution = Double.valueOf(seed + 0.5);
        Object castBroad = bounded.cast(pollution);
        System.out.println("bound-cast=" + (castBroad == pollution) + ",state=" + bounded.state());
        try { bounded.cast(new probe.shadow.Number()); }
        catch (ClassCastException expected) { System.out.println("bound-cast-reject=" + bounded.state()); }
        try {
            Integer castTyped = bounded.cast(pollution);
            System.out.println("unexpected-bound-cast=" + castTyped);
        } catch (ClassCastException expected) { System.out.println("bound-cast-consumer=" + bounded.state()); }
        raw.put(pollution); // Number bridge accepts this; the Integer type argument is erased.
        Object broad = parent.get();
        parent.get(); // Discarding the result must not introduce an Integer checkcast.
        System.out.println("broad=" + (broad == pollution) + ",state=" + bounded.state());
        try {
            Integer typed = parent.get();
            System.out.println("unexpected-read=" + typed);
        } catch (ClassCastException expected) { System.out.println("read-cast=" + bounded.state()); }
        try {
            Integer typed = parent.snapshot();
            System.out.println("unexpected-snapshot=" + typed);
        } catch (ClassCastException expected) { System.out.println("snapshot-cast=" + bounded.state()); }
        Object remembered = raw.snapshot();
        System.out.println("same-pollution=" + (remembered == pollution) + ",state=" + bounded.state());
        bounded.put(null);
        System.out.println("null=" + (parent.get() == null) + ":" + (parent.snapshot() == null)
                + ",state=" + bounded.state());
        bounded.put(original);
        System.out.println("restored=" + (parent.get() == original) + ",state=" + bounded.state());
    }
}
