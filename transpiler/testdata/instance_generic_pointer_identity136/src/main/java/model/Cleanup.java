package model;
public class Cleanup implements AutoCloseable {
    public static String trace = "";
    public static int opens;
    public static int closes;
    public final String name;
    public Cleanup(String name) { this.name = name; opens++; trace += "O" + name; }
    public void close() { closes++; trace += "C" + name; }
}
