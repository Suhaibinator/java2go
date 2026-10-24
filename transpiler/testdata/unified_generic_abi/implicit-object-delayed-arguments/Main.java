package arglate;
class java{static class lang{static class Object{}}}
class Alien{}
class Polluted{<T>T read(){Main.calls++;T result=(T)new Alien();Main.after++;return result;}}
public class Main{
 static int calls,after,effects;
 static <T>T read(){calls++;T result=(T)new Alien();after++;return result;}
 static <A,B>A choose(A first,B second){return first;}
 static int effect(){effects++;return 2;}
 public static void main(String[]args){
  try{String wrong=choose(Main.<String>read(),effect());System.out.println(wrong);}catch(ClassCastException expected){System.out.println("cast:"+calls+":"+after+":"+effects);}
  try{Object wrong=choose(Main.<String>read(),effect());System.out.println(wrong);}catch(ClassCastException expected){System.out.println("broad-cast:"+calls+":"+after+":"+effects);}
  try{String wrong=choose(new Polluted().<String>read(),effect());System.out.println(wrong);}catch(ClassCastException expected){System.out.println("instance:"+calls+":"+after+":"+effects);}
  Object broad=choose(read(),effect());System.out.println("broad:"+(broad instanceof Alien)+":"+calls+":"+after+":"+effects);
  Object empty=choose(null,effect());System.out.println("null:"+(empty==null)+":"+effects);
  int number=choose(9,effect());System.out.println("boxing:"+number+":"+effects);
 }
}
