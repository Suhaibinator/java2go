package app;
public class Main {
    public static void main(java.lang.String[] args) {
        int seed = Integer.parseInt(args[0]);
        origin.Limit value = new origin.Limit(seed);
        Holder<origin.Limit> typed = new Holder<>(value);
        Holder<?> wildcard = typed;
        java.util.ArrayList<java.lang.String> list = new java.util.ArrayList<>();
        list.add("first");
        list.add("second");
        StringHolder<java.util.ArrayList<java.lang.String>> strings = new StringHolder<>(list);
        StringHolder<?> stringWildcard = strings;
        System.out.println(wildcard.read() + ":" + typed.read() + ":" + value.calls + ":" + stringWildcard.size() + ":" + ((Object) wildcard == (Object) typed) + ":" + Holder.Limit.label() + ":" + StringHolder.String.label());
    }
}
