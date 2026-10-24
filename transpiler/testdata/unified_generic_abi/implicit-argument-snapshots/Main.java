package argsnapshot;
class java{static class lang{static class Object{}}}
public class Main{
 static String text="before";static int number=4,effects,bombs;
 static<A,B>A choose(A first,B second){return first;}
 static int effect(){effects++;text="after";number=5;return 2;}
 static String bomb(){bombs++;throw new IllegalStateException("stop");}
 public static void main(String[]args){
  String first=choose(text,effect());System.out.println(first+":"+text+":"+number+":"+effects);
  number=9;int second=choose(number,effect());System.out.println(second+":"+number+":"+effects);
  Object empty=choose(null,effect());System.out.println((empty==null)+":"+effects);
  try{String wrong=choose(bomb(),effect());System.out.println(wrong);}catch(IllegalStateException expected){System.out.println("abrupt:"+bombs+":"+effects);}
 }
}
