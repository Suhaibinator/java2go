package enumfrontier;
public class ArraysControl {
 enum Choice { A { public String marker(){return "a";} }, B; public String marker(){return "b";} }
 enum Other { X }
 enum Empty {;}
 static int effects;
 static Object wrong(){effects++;return Other.X;}
 static Choice[] pass(Choice[] a){return a;}
 static Choice qualifier(){effects++;return null;}
 public static void main(String[] args){
  Choice[] first=Choice.values(), second=Choice.values();
  System.out.println("fresh="+(first!=second)+":"+first.length+":"+(first[0]==Choice.A)+":"+(first[1]==Choice.B));
  first[0]=null; first[1]=Choice.A;
  System.out.println("isolated="+(second[0]==Choice.A)+":"+(Choice.values()[1]==Choice.B)+":"+(Choice.valueOf("B")==Choice.B));
  java.lang.Enum<?>[] enums=second; java.lang.Object[] objects=enums;
  objects[0]=null; objects[1]=Choice.A;
  System.out.println("alias="+(second[0]==null)+":"+(second[1]==Choice.A));
  try{objects[0]=wrong();System.out.println("store=unexpected");}catch(ArrayStoreException expected){System.out.println("store="+effects+":"+(second[0]==null));}
  try{objects[0]="bad";System.out.println("string-store=unexpected");}catch(ArrayStoreException expected){System.out.println("string-store=ArrayStoreException");}
  System.out.println("component="+(Choice.values().getClass().getComponentType()==Choice.class));
  System.out.println("empty="+Empty.values().length+":"+(Empty.values()!=Empty.values()));
  Choice[] copy=Choice.values().clone();copy[0]=Choice.B;
  System.out.println("clone="+(Choice.values()[0]==Choice.A)+":"+(copy[0]==Choice.B));
  String trace="";for(Choice c:Choice.values())trace+=c.name();
  System.out.println("flow="+trace+":"+pass(Choice.values()).length+":"+Choice.values()[0].marker());
  System.out.println("qualifier="+qualifier().values().length+":"+effects);
 }
}
