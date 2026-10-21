package rawret;
import java.util.function.Function;
class Object{}
class java{static class lang{static class Object{}}}
class Alien extends Object{}
class Polluted{<T>T id(T value){Main.calls++;T result=(T)new Alien();Main.after++;return result;}}
interface Echo{String echo(String value);}
public class Main{
 static int calls,after,finals;
 static <T>T read(Object value){calls++;T result=(T)value;after++;return result;}
 public static void main(String[]args){
  Function statik=key->Main.<String>read(new Alien());
  Object first=(Object)statik.apply(null);System.out.println("static:"+(first instanceof Alien)+":"+calls+":"+after);
  try{String wrong=(String)statik.apply(null);System.out.println("wrong:"+wrong);}catch(ClassCastException expected){System.out.println("cast:"+calls+":"+after);}
  Function instance=key->new Polluted().id("x");Object second=(Object)instance.apply(null);System.out.println("instance:"+(second instanceof Alien)+":"+calls+":"+after);
  Function empty=key->Main.<String>read(null);System.out.println("null:"+(empty.apply(null)==null)+":"+calls+":"+after);
  Function nested=key->{try{Function<String,String> typed=value->{return value;};Echo source=value->{return value;};return typed.apply(source.echo("ok"));}finally{finals++;}};
  System.out.println(nested.apply(null)+":"+finals);
 }
}
