package access.observe;
import access.fixture.Target;
import java.lang.reflect.AccessibleObject;
import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Member;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
public final class AccessProbe {
    private static String units(String value) {
        if (value == null) return "null";
        StringBuilder out = new StringBuilder();
        out.append(value.length()).append(':');
        for (int i = 0; i < value.length(); i++) {
            if (i > 0) out.append(',');
            out.append(Integer.toHexString(value.charAt(i)));
        }
        return out.toString();
    }
    private static void failure(String label, Throwable failure) {
        int depth = 0;
        for (Throwable current = failure; current != null; current = current.getCause()) {
            System.out.println(label + ":cause=" + depth + ":class=" + current.getClass().getName()
                + ":message=" + units(current.getMessage()));
            depth++;
        }
    }
    private static void state(String label, AccessibleObject member, Object receiver) {
        try {
            System.out.println(label + ":direct=" + member.canAccess(receiver));
        } catch (Exception failure) {
            failure(label + ":direct", failure);
        }
        try {
            Method checker = AccessibleObject.class.getDeclaredMethod("canAccess", Object.class);
            System.out.println(label + ":reflective=" + checker.invoke(member, receiver));
        } catch (Exception failure) {
            failure(label + ":reflective", failure);
        }
    }
    private static void override(String label, AccessibleObject member, boolean enabled) {
        try {
            member.setAccessible(enabled);
            System.out.println(label + ":set=" + enabled);
        } catch (Exception failure) {
            failure(label, failure);
        }
    }
    private static void bulk(String label, AccessibleObject[] members, boolean enabled) {
        try {
            AccessibleObject.setAccessible(members, enabled);
            System.out.println(label + ":bulk=" + enabled);
        } catch (Exception failure) {
            failure(label, failure);
        }
    }
    private static <M extends AccessibleObject & Member> void intersection(String label, M member, Object receiver) {
        Object checked = Modifier.isStatic(member.getModifiers()) ? null : receiver;
        System.out.println(label + ":kind=" + member.getClass().getName() + ":declaring=" + member.getDeclaringClass().getName());
        state(label, member, checked);
    }
    public static void main(String[] args) throws Exception {
        Target target = Target.fresh();
        Field field = Target.class.getDeclaredField("privateValue");
        Field fieldCopy = Target.class.getDeclaredField("privateValue");
        Field publicField = Target.class.getDeclaredField("publicValue");
        Field staticField = Target.class.getDeclaredField("publicStatic");
        Method method = Target.class.getDeclaredMethod("privateRead");
        Constructor<Target> constructor = Target.class.getDeclaredConstructor();
        intersection("field-before", field, target);
        intersection("method-before", method, target);
        intersection("ctor-before", constructor, null);
        state("public-field", publicField, target);
        state("static-field", staticField, null);
        state("instance-null", field, null);
        state("instance-wrong", field, new Object());
        state("static-nonnull", staticField, target);
        state("ctor-nonnull", constructor, target);
        bulk("source-members", new AccessibleObject[]{field, method, constructor}, true);
        state("field-after", field, target);
        state("field-copy", fieldCopy, target);
        state("method-after", method, target);
        state("ctor-after", constructor, null);
        state("overridden-instance-null", field, null);
        state("overridden-ctor-nonnull", constructor, target);
        System.out.println("override-values=" + field.get(target) + ":" + method.invoke(target)
            + ":" + constructor.newInstance().publicValue);
        override("field-reset", field, false);
        state("field-reset-state", field, target);
        bulk("null-array", null, true);
        bulk("null-entry", new AccessibleObject[]{field, null}, true);
        state("after-null-entry", field, target);
        Field closedModule = String.class.getDeclaredField("value");
        override("closed-module", closedModule, true);
        bulk("mixed-closed-module", new AccessibleObject[]{field, closedModule}, true);
        state("after-mixed-refusal", field, target);
    }
}
