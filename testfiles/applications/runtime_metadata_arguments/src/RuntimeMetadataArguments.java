package parity.metadataarguments;
import java.lang.reflect.InvocationTargetException;
public class RuntimeMetadataArguments {
    public static class Plugin {
        public Plugin() {}
        public String name() { return "plugin"; }
    }
    public static class Broken {
        public Broken() { throw new IllegalStateException("constructor"); }
    }
    public static String run() throws Exception {
        Class<?> type = Class.forName/* registry must follow the AST */("parity.metadataarguments.RuntimeMetadataArguments$Plugin");
        Object plugin = type.getConstructor(new Class<?>[0]).newInstance(new Object[0]);
        String result = "" + type.getMethod("name", new Class<?>[0]).invoke(plugin, new Object[0]);
        result += "/" + type.getMethod("name", (Class<?>[]) null).invoke(plugin, (Object[]) null);
        try { type.getMethod("name").invoke(plugin, (Object) null); }
        catch (IllegalArgumentException error) { result += "/arity"; }
        try { Broken.class.getConstructor().newInstance(); }
        catch (InvocationTargetException error) { result += "/" + error.getCause().getMessage(); }
        return result;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
