package access.api;
public class Owner {
 public static int calls;
 public static String staticValue(){calls++;return "static";}
 public String instanceValue(){calls++;return "instance";}
}
