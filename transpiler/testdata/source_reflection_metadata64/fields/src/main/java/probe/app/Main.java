package probe.app;
import probe.model.State;
import java.lang.reflect.*;
public final class Main {
 public static void main(String[] args)throws Exception{
  int seed=Integer.parseInt(args[0]);State state=new State(seed);
  Field[] fields=State.class.getDeclaredFields();int count=0;
  for(Field field:fields){
   count++;System.out.println("field|"+field.getName()+"|"+field.getModifiers()+"|"+field.getType().getName()+"|"+field.getGenericType().getTypeName()+"|"+(field.getDeclaringClass()==State.class));
  }
  System.out.println("inventory|"+count);
  try{State.class.getDeclaredField("inherited");throw new AssertionError("inherited declared field");}
  catch(NoSuchFieldException failure){System.out.println("inherited|"+failure.getClass().getName());}
  Field hidden=State.class.getDeclaredField("hidden");
  try{hidden.get(state);throw new AssertionError("private access");}
  catch(IllegalAccessException failure){System.out.println("denied|"+failure.getClass().getName());}
  hidden.setAccessible(true);hidden.set(state,"changed-"+seed);
  Field visible=State.class.getDeclaredField("visible");visible.set(state,Integer.valueOf(seed+2));
  Field shared=State.class.getDeclaredField("shared");shared.set(null,Integer.valueOf(seed%7));
  try{State.class.getDeclaredField("FIXED").set(null,Integer.valueOf(seed));throw new AssertionError("static final modified");}
  catch(IllegalAccessException failure){System.out.println("final|"+failure.getClass().getName());}
  System.out.println("state|"+state.snapshot()+"|"+hidden.get(state)+"|"+visible.get(state)+"|"+shared.get(null));
 }
}
