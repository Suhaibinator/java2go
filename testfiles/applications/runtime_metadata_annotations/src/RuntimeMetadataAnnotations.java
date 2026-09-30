package parity.metadataannotations;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Inherited;
public class RuntimeMetadataAnnotations {
    @Retention(RetentionPolicy.RUNTIME)
    public @interface Component {}
    public @interface BuildOnly {}
    @Retention(RetentionPolicy.SOURCE)
    public @interface SourceOnly {}
    @Inherited
    @Retention(RetentionPolicy.RUNTIME)
    public @interface PluginKind {}
    @PluginKind
    public static class Base { public Base() {} }
    @Component @BuildOnly @SourceOnly
    public static class Greeting extends Base {
        public String prefix;
        public Greeting() {}
        public String render() { return prefix + " component"; }
    }
    public static class Ignored { public Ignored() {} }
    public static class Child extends Greeting { public Child() {} }
    public static String run() throws Exception {
        String output = "";
        String[] candidates = {"parity.metadataannotations.RuntimeMetadataAnnotations$Ignored", "parity.metadataannotations.RuntimeMetadataAnnotations$Greeting"};
        for (String name : candidates) {
            Class<?> type = Class.forName(name);
            if (type.isAnnotationPresent(Component.class)) {
                Object instance = type.getConstructor().newInstance();
                type.getField("prefix").set(instance, "hello");
                output += type.getMethod("render").invoke(instance);
            }
        }
        output += "/" + Greeting.class.isAnnotationPresent(BuildOnly.class)
            + "/" + Greeting.class.isAnnotationPresent(SourceOnly.class)
            + "/" + Greeting.class.isAnnotationPresent(PluginKind.class)
            + "/" + Child.class.isAnnotationPresent(Component.class)
            + "/" + Child.class.isAnnotationPresent(PluginKind.class);
        return output;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
