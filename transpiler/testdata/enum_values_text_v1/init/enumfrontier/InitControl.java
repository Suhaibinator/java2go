package enumfrontier;
public class InitControl {
 static String trace="";
 enum Recursive {A,B;
  Recursive(){trace+="ctor:"+name()+":";try{Recursive.values();trace+="unexpected;";}catch(NullPointerException expected){trace+="NPE;";}}
  static String ready=check();static String check(){trace+="ready:"+values().length+";";return "ok";}
 }
 enum Failed {A;static boolean broken=fail();static boolean fail(){throw new IllegalStateException("boom");}}
 static int qualifiers;static Recursive qualifier(){qualifiers++;trace+="qualifier;";return null;}
 public static void main(String[] args){
  System.out.println("before="+trace);
  Recursive[] first=qualifier().values();System.out.println("init="+trace+":"+qualifiers+":"+first.length);
  System.out.println("later="+(Recursive.values()!=first)+":"+trace+":"+Recursive.ready);
  try{Failed.values();System.out.println("first-fail=unexpected");}catch(ExceptionInInitializerError expected){System.out.println("first-fail=ExceptionInInitializerError");}
  try{Failed.values();System.out.println("later-fail=unexpected");}catch(NoClassDefFoundError expected){System.out.println("later-fail=NoClassDefFoundError");}
  System.out.println("cleanup="+Recursive.values().length);
 }
}
