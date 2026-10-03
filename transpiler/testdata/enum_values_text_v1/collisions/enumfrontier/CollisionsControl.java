package enumfrontier;
public class CollisionsControl {
 static String trace="";
 enum Method {A;static int $VALUES(){return 1;}Method(){try{values();}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 enum Constant {$VALUES;Constant(){try{values();}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 enum Member {A;static class $VALUES{}Member(){try{values();}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 enum Repeated {A;static int $VALUES;static int $VALUES$;static int $VALUES$$;Repeated(){try{values();}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 public static void main(String[] args){Method.values();Constant.values();Member.values();Repeated.values();System.out.println("messages="+trace);System.out.println("cleanup="+Method.values().length+":"+Constant.values().length+":"+Member.values().length+":"+Repeated.values().length);}
}
