package probe.app;
import probe.model.State;
import java.lang.reflect.*;
public final class Main {
 public static void main(String[] args)throws Exception{
  int seed=Integer.parseInt(args[0]);Constructor<State> ctor=State.class.getDeclaredConstructor();
  System.out.println("ctor|"+ctor.getModifiers()+"|"+(ctor.getDeclaringClass()==State.class)+"|"+ctor.canAccess(null));
  try{ctor.newInstance();throw new AssertionError("private construction");}
  catch(IllegalAccessException error){System.out.println("access|"+error.getClass().getName());}
  ctor.setAccessible(true);State state=ctor.newInstance();state.apply(seed%11);
  State.rejectNext();
  try{ctor.newInstance();throw new AssertionError("missing invocation cause");}
  catch(InvocationTargetException error){Throwable cause=error.getCause();System.out.println("cause|"+cause.getClass().getName()+"|"+cause.getMessage()+"|"+(cause==State.failure));}
  System.out.println("state|"+ctor.canAccess(null)+"|"+state.revision()+"|"+(ctor.newInstance()!=state));
 }
}
