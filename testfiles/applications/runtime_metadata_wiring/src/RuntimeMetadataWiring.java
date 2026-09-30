package parity.metadatawiring;

import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Method;

public class RuntimeMetadataWiring {
    public static class Service {
        public Service() {}
    }
    @Deprecated
    public static class Greeting extends Service {
        public String prefix = "unset";
        public Greeting() {}
        public String render() { return prefix + " world"; }
    }
    public static String run() throws Exception {
        Class<?> type = Class.forName("parity.metadatawiring.RuntimeMetadataWiring$Greeting");
        Constructor<?> constructor = type.getConstructor();
        Object service = constructor.newInstance();
        Field field = type.getField("prefix");
        field.set(service, "hello");
        Method method = type.getMethod("render");
        return type.getName() + "|" + type.getSuperclass().getName()
            + "|" + Service.class.isAssignableFrom(type)
            + "|" + type.isAnnotationPresent(Deprecated.class)
            + "|" + field.get(service) + "|" + method.invoke(service);
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
