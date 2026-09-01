package parity.lazyclassinit;

public final class LazyClassInitializationApplication {
    static String trace = "";
    static int mainMarker = mark("M");

    private LazyClassInitializationApplication() {
    }

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static void main(String[] args) {
        System.out.println("TRACE=" + trace);
        System.out.println("MARKER=" + mainMarker);
    }
}

final class DormantClass {
    static int dormantMarker = LazyClassInitializationApplication.mark("D");

    private DormantClass() {
    }
}
