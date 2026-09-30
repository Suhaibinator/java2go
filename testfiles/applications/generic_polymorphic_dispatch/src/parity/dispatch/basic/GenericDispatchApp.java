package parity.dispatch.basic;

public class GenericDispatchApp {
    interface Echo { <T> T echo(T value); }
    static class Base implements Echo {
        static int effects;
        public <T> T echo(T value) { effects = effects * 10 + 1; return value; }
        String indirect(String value) { return echo(value); }
    }
    static class Child extends Base {
        public <T> T echo(T value) { effects = effects * 10 + 2; return value; }
    }
    public static String run() {
        Child child = new Child();
        Base base = child;
        Echo service = child;
        String first = base.echo("base");
        String second = service.echo("interface");
        String third = base.indirect("indirect");
        return first + ":" + second + ":" + third + ":" + Base.effects;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
