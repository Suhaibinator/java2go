package app;
import model.State;
public class Main {
 static class Raw implements Comparable {public int compareTo(Object value){return 0;}}
 static <Comparable> Object local(){class Local implements java.lang.Comparable<State>{public int compareTo(State value){return 0;}}return new Local();}
 public static void main(String[] args){
  State state=new State(3);State.Child child=new State.Child(4);Raw raw=new Raw();foreign.Value other=new foreign.Value();
  Object anonymous=new java.lang.Comparable<State>(){public int compareTo(State value){return 0;}};
  Object local=Main.<Object>local();
  System.out.println(java.lang.Comparable.class.isAssignableFrom(State.class)+":"+java.io.Serializable.class.isAssignableFrom(State.class));
  System.out.println(java.lang.Comparable.class.isAssignableFrom(State.Child.class)+":"+parent.Order.class.isAssignableFrom(State.Child.class));
  System.out.println((state instanceof java.lang.Comparable)+":"+(raw instanceof java.lang.Comparable)+":"+(anonymous instanceof java.lang.Comparable)+":"+(local instanceof java.lang.Comparable));
  System.out.println(java.lang.Comparable.class.isAssignableFrom(foreign.Value.class)+":"+foreign.Comparable.class.isAssignableFrom(foreign.Value.class));
  Object[] values=new java.lang.Comparable[2];values[0]=state;values[1]=child;
  try{values[1]=other;System.out.println("bad");}catch(ArrayStoreException expected){System.out.println((values[0]==state)+":"+(values[1]==child));}
  System.out.println(((Object)null) instanceof java.lang.Comparable);
 }
}
