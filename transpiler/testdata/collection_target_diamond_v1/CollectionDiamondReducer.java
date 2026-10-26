package diamondproof;

public class CollectionDiamondReducer {
    public static void main(String[] args) {
        java.util.Collection<String> values = new java.util.ArrayList<>();
        values.add(args[0]);
        String first = values.iterator().next();
        if (!first.equals(args[0]) || values.size() != 1) throw new AssertionError("collection target inference");
        System.out.println(first + ":" + values.size());
    }
}
