package model;

public class State {
    public static int calls;
    public static int effects;
    public static int opened;
    public static int closed;
    public static String trace = "";
    public static void mark(String text) { trace += text; }
    public static int input(int value) { effects++; mark("a"); return value; }
    public static java.lang.Integer missing() { effects++; mark("z"); return null; }
    public static Object wrong(int value) { effects++; mark("w"); return new shadow.Integer(value); }
    public static int failInput() { effects++; mark("x"); throw new IllegalArgumentException("argument"); }
    public static void report(String name) {
        System.out.println(name + ":" + calls + ":" + effects + ":" + opened + ":" + closed + ":" + trace);
        trace = "";
    }
}
