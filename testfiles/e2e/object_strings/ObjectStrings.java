import java.util.ArrayList;
import java.util.List;

public class ObjectStrings {
    static class P {
        int value;

        P(int value) {
            this.value = value;
        }

        public String toString() {
            return "P" + value;
        }
    }

    static class Base {
        public String toString() {
            return "base";
        }
    }

    static class Child extends Base {
        public String toString() {
            return "child";
        }
    }

    static class Locked {
        public synchronized String toString() {
            synchronized (this) {
                return "locked";
            }
        }
    }

    public static void main(String[] args) {
        P value = new P(3);
        System.out.println(value);
        System.out.println("concat " + value);
        System.out.println(String.valueOf(value));

        List<P> values = new ArrayList<P>();
        values.add(value);
        values.add(new P(4));
        System.out.println(values);

        Base erased = new Child();
        System.out.println(erased);

        Locked locked = new Locked();
        synchronized (locked) {
            System.out.println(locked);
        }
    }
}
