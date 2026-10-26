package emptyset;

import java.util.Set;
import static java.util.Collections.emptySet;

class BinderOwner<T> {
    final Set<T> field = java.util.Collections.emptySet();
    Set<T> owner() { return emptySet(); }
    <T> boolean inspect(T methodValue) {
        class Local<T> {
            Set<T> local() { return java.util.Collections.emptySet(); }
            Set<T[]> arrays() { return java.util.Collections.<T[]>emptySet(); }
        }
        Local<Integer> local = new Local<>();
        Set<T> method = emptySet();
        java.lang.Object ownerView = owner();
        java.lang.Object fieldView = field;
        java.lang.Object methodView = method;
        java.lang.Object localView = local.local();
        java.lang.Object arrayView = local.arrays();
        return ownerView == fieldView && ownerView == methodView
                && ownerView == localView && ownerView == arrayView;
    }
}
class T2 {}
class Object {}
class NameOwner<Collections, Set> {
    java.util.Set<Set> local() { return java.util.Collections.<Set>emptySet(); }
}
public class EmptySetBinderControl {
    public static void main(String[] args) {
        BinderOwner<String> outer = new BinderOwner<>();
        java.util.Set<T2> nominal = java.util.Collections.<T2>emptySet();
        java.util.Set<emptyset.Object> object = java.util.Collections.<emptyset.Object>emptySet();
        java.util.Set<java.lang.Object> canonical = java.util.Collections.<java.lang.Object>emptySet();
        java.util.Set<int[]> primitiveArrays = java.util.Collections.<int[]>emptySet();
        java.util.Set<String[]> referenceArrays = java.util.Collections.<String[]>emptySet();
        NameOwner<String,Integer> names = new NameOwner<>();
        java.lang.Object a = nominal;
        java.lang.Object b = object;
        java.lang.Object c = canonical;
        java.lang.Object d = primitiveArrays;
        java.lang.Object e = referenceArrays;
        java.lang.Object f = names.local();
        System.out.println("captured=" + outer.<Long>inspect(Long.valueOf(4)));
        System.out.println("nominal-array-binder=" + (a == b && b == c && c == d && d == e && e == f));
    }
}
