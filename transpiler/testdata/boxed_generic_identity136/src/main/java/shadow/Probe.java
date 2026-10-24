package shadow;

import generic.Identity;

public class Probe {
    public static void run(int seed) {
        Integer source = new Integer(seed);
        Integer same = Identity.identity(source);
        java.lang.Integer builtin = Identity.identity(seed);
        System.out.println("shadow:" + (same == source) + ":" + same.value + ":" + builtin.intValue());
        System.out.println("types:" + same.getClass().getName() + ":" + builtin.getClass().getName());
        Object foreign = builtin;
        try {
            Integer wrong = Identity.polluted(foreign);
            System.out.println("shadow-cast-missed:" + wrong.value);
        } catch (ClassCastException expected) {
            System.out.println("shadow-cast:" + expected.getClass().getSimpleName());
        }
        Integer absent = Identity.identity((Integer)null);
        System.out.println("shadow-null:" + (absent == null));
    }
}
