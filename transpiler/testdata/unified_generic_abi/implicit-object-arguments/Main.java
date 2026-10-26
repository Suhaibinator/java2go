package argabi;
class java {static class lang {static class Object {int marker(){return 7;}}}}
public class Main {
 static int effects;
 static <T>T identity(T value){return value;}
 static <A,B>A choose(A first,B second){return first;}
 static int effect(){effects++;return 2;}
 public static void main(String[]args){
  String text=choose(identity("ok"),effect());
  java.lang.Object own=choose(identity(new java.lang.Object()),effect());
  java.lang.Object empty=choose(identity((java.lang.Object)null),effect());
  System.out.println(text+":"+own.marker()+":"+(empty==null)+":"+effects);
 }
}
