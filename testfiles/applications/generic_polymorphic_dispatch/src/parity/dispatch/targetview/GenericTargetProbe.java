package parity.dispatch.targetview;

public class GenericTargetProbe {
    interface View { int id(); }

    static class First implements View {
        public int id() { return 1; }
    }

    static class Second implements View {
        public int id() { return 2; }
    }

    interface Service { <T> T id(T value); }

    static class Provider implements Service {
        @SuppressWarnings("unchecked")
        public <T> T id(T value) { return (T) new Second(); }
    }

    public static String run() {
        Service service = new Provider();
        View result = service.id(new First());
        return "" + result.id();
    }

    public static void main(String[] args) { System.out.println(run()); }
}
