package defaultorigin;
class java{static class lang{static class Object{}}}
class Alien{}
class Polluted{<T>T read(){Main.calls++;T result=(T)new Alien();Main.after++;return result;}}
public class Main{
 static int calls,after;
 static <T>T read(){calls++;T result=(T)new Alien();after++;return result;}
 public static void main(String[]args){
  var first=Main.read();System.out.println("static:"+(first instanceof Alien)+":"+calls+":"+after);
  var second=new Polluted().read();System.out.println("instance:"+(second instanceof Alien)+":"+calls+":"+after);
 }
}
