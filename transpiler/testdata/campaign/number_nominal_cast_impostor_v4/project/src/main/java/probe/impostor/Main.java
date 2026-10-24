package probe.impostor;

import probe.api.Carrier;
import probe.api.Factory;
import probe.model.IntegerCarrier;
import probe.model.NamedFactory;
import probe.model.NumberCarrier;
import probe.model.TextCarrier;

public final class Main {
    private static int argumentCalls;
    private static Object argument(Object value) { argumentCalls++; return value; }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        System.out.println("seed=" + seed);
        Integer original = Integer.valueOf(seed);
        IntegerCarrier concrete = new IntegerCarrier();
        NumberCarrier<Integer> bounded = concrete;
        Carrier<Integer> parent = concrete;
        Carrier raw = concrete;
        bounded.put(original);

        NamedFactory named = new NamedFactory();
        Factory factory = named;
        Carrier<Integer> retained = factory.retain(parent);
        TextCarrier text = new TextCarrier();
        String marker = "factory-" + seed;
        text.put(marker);
        Carrier<String> retainedText = factory.retain(text);
        Carrier<Integer> absent = factory.retain((Carrier<Integer>) null);
        System.out.println("factory=" + (retained == parent) + ":" + (retainedText == text)
                + ":" + (retainedText.get() == marker) + ":" + (absent == null)
                + ":" + (raw == parent) + ",calls=" + named.calls());

        NumberImpostor impostor = new NumberImpostor();
        Object methodShadow = bounded.shadow(impostor);
        probe.shadow.Number sourceNumber = new probe.shadow.Number();
        boolean sourceClassAccepted = SourceNumberCast.cast(sourceNumber) == sourceNumber;
        boolean sourceClassRejected = false;
        try { SourceNumberCast.cast(impostor); }
        catch (ClassCastException expected) { sourceClassRejected = true; }
        System.out.println("ownership-guards=" + (methodShadow == impostor) + ":"
                + sourceClassAccepted + ":" + sourceClassRejected + ",numeric=" + impostor.calls());
        int cleanup = 0;
        try {
            boolean rejected = false;
            try { raw.put(impostor); }
            catch (ClassCastException expected) { rejected = true; }
            Object current = parent.get();
            Object remembered = raw.snapshot();
            System.out.println("raw-impostor=" + rejected + ":" + (current == original)
                    + ":" + (remembered == original) + ",state=" + bounded.state()
                    + ",numeric=" + impostor.calls());
        } finally {
            cleanup++;
            bounded.put(original);
            Object restored = parent.get();
            System.out.println("cleanup.raw=" + cleanup + ":" + (restored == original)
                    + ",state=" + bounded.state() + ",numeric=" + impostor.calls());
        }

        boolean objectRejected = false;
        try { Object wrong = bounded.cast(impostor); System.out.println("unexpected.object=" + (wrong == impostor)); }
        catch (ClassCastException expected) { objectRejected = true; }
        finally { cleanup++; }
        System.out.println("cast.object=" + objectRejected + ",state=" + bounded.state()
                + ",cleanup=" + cleanup + ",numeric=" + impostor.calls());

        boolean discardedRejected = false;
        try { bounded.cast(impostor); }
        catch (ClassCastException expected) { discardedRejected = true; }
        finally { cleanup++; }
        System.out.println("cast.discarded=" + discardedRejected + ",state=" + bounded.state()
                + ",cleanup=" + cleanup + ",numeric=" + impostor.calls());

        boolean typedRejected = false;
        try { Integer wrong = bounded.cast(impostor); System.out.println("unexpected.typed=" + wrong); }
        catch (ClassCastException expected) { typedRejected = true; }
        finally { cleanup++; }
        System.out.println("cast.typed=" + typedRejected + ",state=" + bounded.state()
                + ",cleanup=" + cleanup + ",numeric=" + impostor.calls());

        boolean singleRejected = false;
        try { Object wrong = bounded.cast(argument(impostor)); System.out.println("unexpected.single=" + (wrong == impostor)); }
        catch (ClassCastException expected) { singleRejected = true; }
        finally { cleanup++; }
        System.out.println("cast.single=" + singleRejected + ":" + argumentCalls + ",state=" + bounded.state()
                + ",cleanup=" + cleanup + ",numeric=" + impostor.calls());

        Object afterCasts = parent.get();
        Object rememberedAfterCasts = raw.snapshot();
        System.out.println("retained=" + (afterCasts == original) + ":" + (rememberedAfterCasts == original)
                + ",state=" + bounded.state() + ",numeric=" + impostor.calls());

        Double valid = Double.valueOf(seed + 0.5);
        Object accepted = bounded.cast(valid);
        bounded.cast(valid);
        boolean consumerRejected = false;
        try { Integer narrow = bounded.cast(valid); System.out.println("unexpected.narrow=" + narrow); }
        catch (ClassCastException expected) { consumerRejected = true; }
        finally { cleanup++; }
        System.out.println("valid-number=" + (accepted == valid) + ":" + consumerRejected
                + ",state=" + bounded.state() + ",cleanup=" + cleanup + ",numeric=" + impostor.calls());

        Object nullable = bounded.cast(null);
        System.out.println("null-cast=" + (nullable == null) + ",state=" + bounded.state()
                + ",numeric=" + impostor.calls());

        try {
            raw.put(valid);
            Object broad = parent.get();
            parent.get();
            System.out.println("raw-valid=" + (broad == valid) + ",state=" + bounded.state());
            boolean narrowRejected = false;
            try { Integer narrow = parent.get(); System.out.println("unexpected.read=" + narrow); }
            catch (ClassCastException expected) { narrowRejected = true; }
            System.out.println("read-consumer=" + narrowRejected + ",state=" + bounded.state());
        } finally {
            cleanup++;
            bounded.put(original);
            Object restored = parent.get();
            Object rememberedRestored = raw.snapshot();
            System.out.println("cleanup.valid=" + cleanup + ":" + (restored == original)
                    + ":" + (rememberedRestored == original) + ",state=" + bounded.state()
                    + ",numeric=" + impostor.calls());
        }
    }
}
