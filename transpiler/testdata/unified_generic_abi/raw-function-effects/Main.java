import java.util.function.Function;
interface Echo{String echo(String value);}
public class Main{
 static Object monitor=new Object();static int receivers,arguments,finals;
 static Function receiver(Function value){receivers++;return value;}
 static Object argument(){arguments++;return Integer.valueOf(9);}
 public static void main(String[]args){
  Function value=(Object key)->{synchronized(monitor){if(key==null)return null;return 17;}};
  Object observed=receiver(value).apply(argument());System.out.println(observed+":"+receivers+":"+arguments);
  synchronized(monitor){System.out.println("null:"+(value.apply(null)==null));}
  Function outer=key->{Function<String,String> nested=text->text;Echo source=text->text;try{return nested.apply(source.echo("ok"));}finally{finals++;}};
  System.out.println(outer.apply(null)+":"+finals);
  Function abrupt=key->{try{throw new IllegalArgumentException("bad");}finally{finals++;}};
  try{abrupt.apply(null);System.out.println("unexpected");}catch(IllegalArgumentException expected){System.out.println(expected.getMessage()+":"+finals);}
 }
}
