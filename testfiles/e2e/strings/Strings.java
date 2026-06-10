public class Strings {
    public static void main(String[] args) {
        String s = "Hello, World";
        System.out.println(s.length());
        System.out.println(s.toUpperCase());
        System.out.println(s.toLowerCase());
        System.out.println(s.substring(7));
        System.out.println(s.substring(0, 5));
        System.out.println(s.charAt(1));
        System.out.println(s.indexOf("World"));
        System.out.println(s.replace("World", "Go"));
        System.out.println(s.contains("World"));
        System.out.println("  trim me  ".trim());
        System.out.println("a,b,c".split(",").length);
        System.out.println(s.equals("Hello, World"));
        System.out.println(s.startsWith("Hello"));
        System.out.println(s.endsWith("World"));

        StringBuilder sb = new StringBuilder();
        sb.append("x");
        sb.append("y");
        sb.append(123);
        System.out.println(sb.toString());

        System.out.println(String.valueOf(42));
        System.out.println("n=" + 7 + 8);
    }
}
