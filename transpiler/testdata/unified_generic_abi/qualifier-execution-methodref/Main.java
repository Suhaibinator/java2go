import java.util.function.Function;
public class Main {
 static String trace="";
 static final java.lang.Object lock=new java.lang.Object();
 static final ThreadLocal<String> local=new ThreadLocal<String>();
 static Thread owner;
 static boolean context;
 static final RuntimeException marker=new RuntimeException("abrupt");
 static Main qualifier(boolean fail){trace+="Q";if(fail)throw marker;return null;}
 static java.lang.Object argument(java.lang.Object value,boolean fail){trace+="A";if(fail)throw marker;return value;}
 static <T> T read(java.lang.Object value){trace+="B";context=Thread.currentThread()==owner&&"ctx".equals(local.get())&&Thread.holdsLock(lock);try{T result=(T)value;trace+="R";return result;}catch(ClassCastException e){trace+="I";return null;}finally{trace+="F";}}
 public static void main(String[]args){owner=Thread.currentThread();local.set("ctx");java.lang.Object value=Integer.valueOf(7);
  synchronized(lock){
   trace="";java.lang.Object seen=qualifier(false).<String>read(argument(value,false));System.out.println("object:"+(seen==value)+":"+trace+":"+context);
   trace="";try{String bad=qualifier(false).<String>read(argument(value,false));System.out.println("unexpected:"+bad);}catch(ClassCastException e){trace+="C";}System.out.println("string:"+trace+":"+context);
   Function<java.lang.Object,java.lang.Object> objectRef=Main::<String>read;
   Function<java.lang.Object,String> stringRef=Main::<String>read;
   trace="";java.lang.Object ref=objectRef.apply(value);System.out.println("ref-object:"+(ref==value)+":"+trace+":"+context);
   trace="";try{String bad=stringRef.apply(value);System.out.println("unexpected-ref:"+bad);}catch(ClassCastException e){trace+="C";}System.out.println("ref-string:"+trace+":"+context);
   trace="";try{qualifier(true).<String>read(argument(value,false));}catch(RuntimeException e){System.out.println("qualifier:"+(e==marker)+":"+trace);}
   trace="";try{qualifier(false).<String>read(argument(value,true));}catch(RuntimeException e){System.out.println("argument:"+(e==marker)+":"+trace);}
  }
 }
}
