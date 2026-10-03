package probe.enums;

import java.lang.reflect.Field;
import probe.api.Carrier;
import probe.api.Factory;
import probe.model.NamedFactory;
import probe.model.TextCarrier;
import probe.model.Choice;
import probe.model.ChoiceCarrier;
import probe.model.EnumCarrier;
import probe.model.Other;

public final class Main {
    private static <E extends java.lang.Enum<E>> String inspect(E value) {
        synchronized (value) {
            return value.name() + ":" + value.ordinal() + ":"
                    + value.getDeclaringClass().getName() + ":" + value.toString();
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) throws Exception {
        int seed = Integer.parseInt(args[0]);
        ChoiceCarrier concrete = new ChoiceCarrier();
        EnumCarrier<Choice> bounded = concrete;
        Carrier<Choice> parent = concrete;
        Carrier raw = concrete;
        bounded.put(Choice.SPECIAL);
        System.out.println("identity=" + (parent.get() == Choice.SPECIAL) + ":"
                + (parent.snapshot() == Choice.SPECIAL) + ",state=" + bounded.state());
        String marker = "enum-binder-" + seed;
        System.out.println("method-shadow=" + (bounded.shadow(marker) == marker));
        NamedFactory namedFactory = new NamedFactory();
        Factory factory = namedFactory;
        Carrier<Choice> retained = factory.retain(parent);
        TextCarrier text = new TextCarrier();
        String factoryMarker = "factory-" + seed;
        text.put(factoryMarker);
        Carrier<String> retainedText = factory.retain(text);
        Carrier<Choice> absent = factory.retain((Carrier<Choice>) null);
        Carrier<Choice> direct = namedFactory.retain(parent);
        System.out.println("factory=" + (retained == parent) + ":" + (retainedText == text)
                + ":" + (retainedText.get() == factoryMarker) + ":" + (absent == null)
                + ":" + (direct == parent) + ",calls=" + namedFactory.calls());
        System.out.println("inspect=" + inspect(Choice.SPECIAL) + ",calls=" + Choice.SPECIAL.calls());
        System.out.println("enum-flags=" + java.lang.Enum.class.isEnum() + ":"
                + Choice.class.isEnum() + ":" + Choice.SPECIAL.getClass().isEnum()
                + ",ancestry=" + java.lang.Enum.class.isAssignableFrom(Choice.class)
                + ":" + Choice.class.isAssignableFrom(Choice.SPECIAL.getClass())
                + ",distinct=" + (Choice.SPECIAL.getClass() != Choice.class));
        Field field = Choice.class.getDeclaredField("SPECIAL");
        Field stateField = Choice.class.getDeclaredField("calls");
        System.out.println("fields=" + field.isEnumConstant() + ":" + stateField.isEnumConstant()
                + ":" + (field.get(null) == Choice.SPECIAL));

        try { raw.put("not-enum-" + seed); }
        catch (ClassCastException expected) { System.out.println("reject-string=" + bounded.state()); }
        try { raw.put(new probe.shadow.Enum()); }
        catch (ClassCastException expected) { System.out.println("reject-shadow=" + bounded.state()); }
        try { ((java.lang.Enum) Choice.SPECIAL).compareTo(Other.ONLY); }
        catch (ClassCastException expected) { System.out.println("reject-other-family=" + bounded.state()); }
        System.out.println("compare=" + Choice.FIRST.compareTo(Choice.SPECIAL));

        Object castBroad = bounded.cast(Other.ONLY);
        System.out.println("bound-cast=" + (castBroad == Other.ONLY) + ",state=" + bounded.state());
        try { bounded.cast(new probe.shadow.Enum()); }
        catch (ClassCastException expected) { System.out.println("bound-cast-reject=" + bounded.state()); }
        try {
            Choice castTyped = bounded.cast(Other.ONLY);
            System.out.println("unexpected-bound-cast=" + castTyped);
        } catch (ClassCastException expected) { System.out.println("bound-cast-consumer=" + bounded.state()); }
        raw.put(Other.ONLY); // Erased Enum bridge accepts another enum family.
        Object broad = parent.get();
        parent.get();
        System.out.println("broad=" + (broad == Other.ONLY) + ",state=" + bounded.state());
        try {
            Choice typed = parent.get();
            System.out.println("unexpected-read=" + typed);
        } catch (ClassCastException expected) { System.out.println("read-cast=" + bounded.state()); }
        try {
            Choice typed = parent.snapshot();
            System.out.println("unexpected-snapshot=" + typed);
        } catch (ClassCastException expected) { System.out.println("snapshot-cast=" + bounded.state()); }
        System.out.println("same-pollution=" + (raw.snapshot() == Other.ONLY) + ",state=" + bounded.state());
        bounded.put(null);
        System.out.println("null=" + (parent.get() == null) + ":" + (parent.snapshot() == null)
                + ",state=" + bounded.state());
        bounded.put(Choice.FIRST);
        System.out.println("restored=" + inspect(parent.get()) + ",state=" + bounded.state());
    }
}
