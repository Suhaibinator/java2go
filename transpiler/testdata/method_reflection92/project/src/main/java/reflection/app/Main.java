package reflection.app;
import java.lang.reflect.Method;
import java.lang.reflect.InvocationTargetException;
import reflection.api.*;
import reflection.impl.*;
public class Main {
    static void describe(String label, Method method, Class<?> result) {
        System.out.println(label + ":" + method.getName() + ":" + method.isBridge() + ":" + method.isSynthetic()
            + ":" + method.getReturnType().getName() + ":" + (method.getReturnType() == result)
            + ":" + method.getDeclaringClass().getName());
    }
    public static void main(String[] args) throws Exception {
        Text text = new Text();
        Method echo = Text.class.getMethod("echo", Object.class);
        describe("erased", echo, Object.class);
        describe("narrow", Text.class.getMethod("echo", String.class), String.class);
        describe("value", Text.class.getMethod("value"), String.class);
        describe("inherited", Text.class.getMethod("unchanged"), String.class);
        describe("primitive", Text.class.getMethod("count"), int.class);
        describe("hidden", Base.class.getDeclaredMethod("hidden"), String.class);
        int bridges = 0, real = 0;
        for (Method method : Text.class.getDeclaredMethods()) {
            if (method.isBridge() && method.isSynthetic()) bridges++;
            else if (!method.isBridge() && !method.isSynthetic()) real++;
        }
        System.out.println("declared:" + bridges + ":" + real);
        System.out.println("invoke:" + echo.invoke(text, "z") + ":" + Base.class.getMethod("value").invoke(new Revision()));
        try { echo.invoke(text, Integer.valueOf(7)); System.out.println("wrong-bridge"); }
        catch (InvocationTargetException expected) { System.out.println("bridge-cause:" + expected.getCause().getClass().getName()); }
        System.out.println("effects:" + text.calls);
        try { Text.class.getDeclaredMethod("unchanged"); System.out.println("wrong-inherited"); }
        catch (NoSuchMethodException expected) { System.out.println("declared-only"); }
        describe("covariant-class", NamedLeaf.class.getMethod("kind"), Child.class);
        describe("covariant-interface", NamedImplementation.class.getMethod("kind"), Child.class);
        System.out.println("covariant-invoke:" + ((Root) Named.class.getMethod("kind").invoke(new NamedLeaf())).tag());
        Method parent = Named.class.getMethod("kind");
        System.out.println("class-identity:" + (parent.getReturnType() == Root.class));
    }
}
