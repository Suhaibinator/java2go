package example.domain;
import vendor.text.Formatter;
public class Invoice { public static String render(String name, int quantity) { return Formatter.format(name, quantity * 7); } }
