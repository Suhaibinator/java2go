package anonymouscollision;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.Set;

class Box<receiver,execution,other> {
    Set<String> create() {
        final ArrayList<String> values = new ArrayList<>(); values.add("item");
        return new AbstractSet<String>() {
            public int size() {
                AnonymousBinderCollisionControl.context();
                if (!equals("source") || hashCode(19) != 19) throw new AssertionError("source overloads");
                return values.size();
            }
            public Iterator<String> iterator() { AnonymousBinderCollisionControl.context(); return values.iterator(); }
            public boolean equals(String value) {
                AnonymousBinderCollisionControl.stringCalls++;
                return "source".equals(value);
            }
            public int hashCode(int value) {
                AnonymousBinderCollisionControl.intCalls++;
                return value;
            }
        };
    }
}
public class AnonymousBinderCollisionControl {
    static final ThreadLocal<String> local = new ThreadLocal<>();
    static final Object lock = new Object();
    static Thread owner;
    static int stringCalls,intCalls;
    static boolean sameExecution = true;
    static void context() {
        sameExecution &= Thread.currentThread() == owner && Thread.holdsLock(lock) && "caller".equals(local.get());
    }
    public static void main(String[] args) {
        owner = Thread.currentThread(); local.set("caller");
        synchronized(lock) {
            Set<String> first = new Box<String,String,String>().create();
            Set<String> second = new Box<String,String,String>().create();
            Object object = first;
            boolean equality = object.equals(first) && object.equals(second) && second.equals(object);
            boolean hashing = object.hashCode() == second.hashCode();
            System.out.println("object-defaults=" + (equality && hashing));
            System.out.println("source-overloads=" + (stringCalls > 0 && intCalls == stringCalls));
            System.out.println("same-execution=" + sameExecution);
        }
    }
}
