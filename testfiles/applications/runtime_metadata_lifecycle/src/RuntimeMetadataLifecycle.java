package parity.metadatalifecycle;

import java.lang.reflect.InvocationTargetException;
public class RuntimeMetadataLifecycle {
    static String events = "";
    public static class Base {
        public String label = "base";
        public Base() {}
        public String describe() { return "base"; }
    }
    public static class Plugin extends Base {
        static { events += "I"; }
        public String label = "child";
        public int count = 7;
        public long wide = 0;
        public Base reference;
        public final int fixed = 3;
        public Plugin() { events += "C"; }
        public synchronized String describe() { return "plugin"; }
        public void fail() { throw new IllegalStateException("target"); }
    }
    public static String run() throws Exception {
        Class<?> literal = Plugin.class;
        String before = events;
        Class<?> found = Class.forName("parity.metadatalifecycle.RuntimeMetadataLifecycle$Plugin");
        String loaded = events;
        Object plugin = found.getConstructor().newInstance();
        String result = before + "/" + loaded + "/" + events + "/" + (literal == found);
        result += "/" + Base.class.getField("label").get(plugin);
        result += "/" + found.getField("label").get(plugin);
        found.getField("count").set(plugin, 9);
        result += "/" + found.getField("count").get(plugin);
        found.getField("wide").set(plugin, 12);
        found.getField("count").set(plugin, Byte.valueOf((byte) 2));
        found.getField("label").set(plugin, null);
        found.getField("reference").set(plugin, plugin);
        result += "/" + found.getField("wide").get(plugin) + "/" + found.getField("count").get(plugin)
            + "/" + (found.getField("label").get(plugin) == null)
            + "/" + (found.getField("reference").get(plugin) == plugin);
        found.getField("reference").set(plugin, null);
        result += "/" + (found.getField("reference").get(plugin) == null);
        synchronized (plugin) { result += "/" + Base.class.getMethod("describe").invoke(plugin); }
        try { found.getMethod("fail").invoke(plugin); }
        catch (InvocationTargetException error) { result += "/" + error.getCause().getMessage(); }
        try { Class.forName("missing.Plugin"); }
        catch (ClassNotFoundException error) { result += "/missing"; }
        try { found.getField("absent"); }
        catch (NoSuchFieldException error) { result += "/field"; }
        try { found.getMethod("describe", String.class); }
        catch (NoSuchMethodException error) { result += "/method"; }
        try { found.getField("fixed").set(plugin, Integer.valueOf(4)); }
        catch (IllegalAccessException error) { result += "/final"; }
        return result;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
