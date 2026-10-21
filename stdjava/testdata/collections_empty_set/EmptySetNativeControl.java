package emptysetnative;

import java.util.*;
import java.util.function.*;

public class EmptySetNativeControl {
 static String trace="";
 static Throwable failure(Runnable action) {
  try { action.run(); } catch(Throwable failure) { return failure; }
  throw new AssertionError("missing expected failure");
 }
 static void diagnostic(String key,Runnable action) {
  Throwable caught=failure(action);
  System.out.println(key+"="+caught.getClass().getName()+":"+String.valueOf(caught.getMessage()));
 }
 static class Probe extends AbstractSet<Object> {
  int size; boolean empty; RuntimeException sizeFailure,emptyFailure;
  final List<Object> elements=new ArrayList<>();
  Probe(int size,boolean empty,Object... values){this.size=size;this.empty=empty;elements.addAll(Arrays.asList(values));}
  public int size(){trace+="S";if(sizeFailure!=null)throw sizeFailure;return size;}
  public boolean isEmpty(){trace+="E";if(emptyFailure!=null)throw emptyFailure;return empty;}
  public Iterator<Object> iterator(){
   trace+="I";return new Iterator<Object>(){
    int index;
    public boolean hasNext(){trace+="H";return index<elements.size();}
    public Object next(){trace+="N";return elements.get(index++);}
    public void remove(){trace+="R";throw new AssertionError("iterator.remove");}
   };
  }
  public boolean contains(Object value){trace+="C";throw new AssertionError("contains");}
 }
 @SuppressWarnings({"rawtypes","unchecked"})
 public static void main(String[] args) {
  Set raw=Collections.emptySet();
  Object[] a=raw.toArray(),b=raw.toArray();
  String[] zero=new String[0],tail={"first","tail"};
  boolean arrays=a!=b&&a.length==0&&a.getClass()==Object[].class
    &&raw.toArray(zero)==zero&&raw.toArray(tail)==tail&&tail[0]==null&&tail[1].equals("tail");
  int[] kept={2};int[][] ranks={{1},kept};String[] keptStrings={"last"};String[][] strings={{"first"},keptStrings};
  boolean rank=raw.toArray(ranks)==ranks&&ranks[0]==null&&ranks[1]==kept&&ranks.getClass()==int[][].class
    &&raw.toArray(strings)==strings&&strings[0]==null&&strings[1]==keptStrings&&strings.getClass()==String[][].class;
  System.out.println("arrays="+arrays+":"+rank);
  diagnostic("containsAllNull",()->raw.containsAll(null));
  diagnostic("toArrayNull",()->raw.toArray((Object[])null));
  diagnostic("retainAllNull",()->raw.retainAll(null));
  diagnostic("removeAllNull",()->raw.removeAll(null));
  diagnostic("forEachNull",()->raw.forEach(null));
  diagnostic("removeIfNull",()->raw.removeIf(null));
  diagnostic("iteratorForEachNull",()->raw.iterator().forEachRemaining(null));
  diagnostic("addNull",()->raw.add(null));
  diagnostic("next",()->raw.iterator().next());
  diagnostic("iteratorRemove",()->raw.iterator().remove());
  Throwable first=failure(()->raw.add(null)),second=failure(()->raw.add(null));
  first.initCause(null);boolean cause=true;try{first.initCause(null);cause=false;}catch(IllegalStateException expected){}
  System.out.println("freshException="+(first!=second)+":"+(first.getMessage()==null)+":"+cause);
  trace="";boolean addEmpty=raw.addAll(new Probe(91,false));System.out.println("addEmpty="+addEmpty+":"+trace);
  trace="";Throwable addFailure=failure(()->raw.addAll(new Probe(0,true,new Object())));
  System.out.println("addNonempty="+addFailure.getClass().getName()+":"+trace);
  trace="";boolean retain=raw.retainAll(new Probe(-1,false,new Object()));System.out.println("retain="+retain+":"+trace);
  trace="";boolean remove=raw.removeAll(new Probe(1,false,new Object()));System.out.println("removeRegular="+remove+":"+trace);
  trace="";boolean negative=raw.removeAll(new Probe(-1,false,new Object()));System.out.println("removeNegative="+negative+":"+trace);
  Probe eq=new Probe(0,true);trace="";System.out.println("equals="+raw.equals(eq)+":"+trace);
  eq.empty=false;trace="";System.out.println("equalsNotEmpty="+raw.equals(eq)+":"+trace);
  ClassCastException sizeFailure=new ClassCastException("size sentinel");eq.sizeFailure=sizeFailure;trace="";
  Throwable observed=failure(()->raw.equals(eq));System.out.println("sizeBoundary="+(observed==sizeFailure)+":"+trace);
  eq.sizeFailure=null;eq.emptyFailure=new ClassCastException("isEmpty sentinel");trace="";
  System.out.println("containsBoundary="+raw.equals(eq)+":"+trace);
  eq.emptyFailure=new NullPointerException("isEmpty sentinel");trace="";
  System.out.println("nullBoundary="+raw.equals(eq)+":"+trace);
  eq.emptyFailure=new IllegalArgumentException("isEmpty sentinel");trace="";observed=failure(()->raw.equals(eq));
  System.out.println("otherBoundary="+(observed==eq.emptyFailure)+":"+trace);
  int[] calls={0};Consumer<Object> consumer=v->{calls[0]++;throw new AssertionError("consumer");};Predicate<Object> predicate=v->{calls[0]++;throw new AssertionError("predicate");};
  raw.forEach(consumer);raw.iterator().forEachRemaining(consumer);boolean removed=raw.removeIf(predicate);
  System.out.println("functional="+(calls[0]==0)+":"+removed);
  Set<Object> nativeEmpty=new HashSet<>();Set<Object> nativeFull=new HashSet<>();nativeFull.add(new Object());
  System.out.println("legacyEquality="+raw.equals(nativeEmpty)+":"+nativeEmpty.equals(raw)+":"+raw.equals(nativeFull)+":"+raw.hashCode()+":"+raw.toString());
 }
}
