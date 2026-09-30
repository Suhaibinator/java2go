package parity.dispatch.timing;

public class GenericDispatchTimingApp {
    interface Service { <T> T echo(T value); }
    interface CastService { <T> T unchecked(Object value); }
    interface Expected { int value(); }
    static class Impostor { public int value() { return 9; } }
    static class Base implements Service, CastService {
        @SuppressWarnings("unchecked")
        public <T> T unchecked(Object value) { effects = effects * 10 + 4; return (T) value; }
        public <T> T echo(T value) { effects = effects * 10 + 1; return value; }
    }
    static class Polluted extends Base {
        @SuppressWarnings("unchecked")
        public <T> T echo(T value) { effects = effects * 10 + 2; return (T) new Object(); }
    }
    static int effects;
    static String argument() { effects = effects * 10 + 3; return "value"; }
    public static String run() {
        Service service = new Polluted();
        service.echo("discarded");
        String outcome;
        try { String result = service.echo(argument()); outcome = result; }
        catch (ClassCastException expected) { outcome = "cast"; }
        Base missing = null;
        try { missing.echo(argument()); outcome += ":missing"; }
        catch (NullPointerException expected) { outcome += ":null"; }
        Service absent = null;
        try { absent.echo(argument()); outcome += ":missing"; }
        catch (NullPointerException expected) { outcome += ":null"; }
        CastService casts = new Polluted();
        try { Expected result = casts.unchecked(new Impostor()); outcome += ":accepted" + result.value(); }
        catch (ClassCastException expected) { outcome += ":nominal"; }
        return outcome + ":" + effects;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
