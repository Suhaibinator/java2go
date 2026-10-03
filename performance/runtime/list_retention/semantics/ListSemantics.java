import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

public final class ListSemantics {
    public static String run() {
        Object a = new Object();
        Object b = new Object();
        List<Object> refs = new ArrayList<>();
        refs.add(a); refs.add(null); refs.add(b);
        boolean setIdentity = refs.set(1, a) == null;
        boolean removedIdentity = refs.remove(0) == a;
        boolean removedObject = refs.remove(b);
        boolean survivorIdentity = refs.get(0) == a;
        refs.add(null);
        boolean removedNull = refs.remove(null);
        boolean absentNull = !refs.remove(null);
        boolean tailIdentity = refs.remove(0) == a;
        boolean empty = refs.isEmpty();
        String bounds = "";
        try { refs.remove(0); } catch (IndexOutOfBoundsException e) { bounds += "E"; }
        refs.add(a);
        try { refs.remove(-1); } catch (IndexOutOfBoundsException e) { bounds += "N"; }
        try { refs.remove(1); } catch (IndexOutOfBoundsException e) { bounds += "H"; }
        List<String> strings = new ArrayList<>();
        strings.add("a"); strings.add(null); strings.add("b"); strings.add("a");
        boolean duplicate = strings.remove("a");
        String before = "";
        for (String value : strings) { before += value == null ? "N" : value; }
        boolean nullSet = strings.set(0, "x") == null;
        String during = "";
        for (String value : strings) {
            during += value;
            if (value.equals("x")) { strings.set(1, "z"); }
        }
        Object[] backing = new Object[] { a, b };
        List<Object> fixed = Arrays.asList(backing);
        boolean writeThrough = fixed.set(1, null) == b && backing[1] == null;
        String fixedErrors = "";
        try { fixed.remove(0); } catch (UnsupportedOperationException e) { fixedErrors += "R"; }
        try { fixed.remove(99); } catch (UnsupportedOperationException e) { fixedErrors += "I"; }
        try { fixed.remove(a); } catch (UnsupportedOperationException e) { fixedErrors += "O"; }
        boolean fixedMissing = !fixed.remove(b);
        String fixedIteration = "";
        for (Object value : fixed) {
            fixedIteration += value == null ? "N" : "A";
            if (value == a) { backing[1] = b; }
        }
        return setIdentity + ":" + removedIdentity + ":" + removedObject + ":" + survivorIdentity
                + ":" + removedNull + ":" + absentNull + ":" + tailIdentity + ":" + empty
                + ":" + bounds + ":" + duplicate + ":" + before + ":" + nullSet + ":" + during
                + ":" + writeThrough + ":" + fixedErrors + ":" + fixedMissing + ":" + fixedIteration;
    }
}
