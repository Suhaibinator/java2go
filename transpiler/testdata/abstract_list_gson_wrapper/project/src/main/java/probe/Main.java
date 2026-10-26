package probe;
import java.util.*;import com.google.gson.internal.NonNullElementWrapperList;
public class Main {
 static ArrayList<Object> values(){ArrayList<Object> a=new ArrayList<>();a.add("a");a.add("b");a.add("c");return a;}
 static class Plain extends AbstractList<Object> {
  ArrayList<Object> data=values();int gets,sizes,sets,adds,removes;
  public Object get(int index){gets++;return data.get(index);}
  public int size(){sizes++;return data.size();}
  public Object set(int index,Object value){sets++;return data.set(index,value);}
  public void add(int index,Object value){adds++;data.add(index,value);modCount++;}
  public Object remove(int index){removes++;Object old=data.remove(index);modCount++;return old;}
  int generation(){return modCount;}
 }
 public static void main(String[]args){
  ArrayList<Object> d=values();NonNullElementWrapperList<Object> w=new NonNullElementWrapperList<>(d);List<Object> a=w;Collection<Object> c=w;Object object=w;
  System.out.println("roles:"+(a==object)+":"+(c==object)+":"+(object instanceof RandomAccess)+":"+(object instanceof AbstractList)+":"+w.getClass().getSuperclass().getName());
  System.out.println("set:"+a.set(1,"B")+":"+d+":"+a.size()+":"+a.contains(null)+":"+a.add("d"));
  try{a.set(99,null);System.out.println("bad-null");}catch(NullPointerException e){System.out.println("null:"+e.getMessage()+":"+d.size());}
  Iterator<Object> it=a.iterator();System.out.println("first:"+it.next());d.add("e");System.out.println("delegate-edit:"+it.next()+":"+it.hasNext());it.remove();System.out.println("iterator-remove:"+d+":"+it.next());
  while(it.hasNext())it.next();try{it.next();System.out.println("bad-end");}catch(NoSuchElementException e){System.out.println("end:"+(e.getCause() instanceof IndexOutOfBoundsException));}
  List<Object> sub=a.subList(1,3);System.out.println("sub:"+sub+":"+(((Object)sub) instanceof RandomAccess));d.add(0,"z");System.out.println("shift:"+sub.get(0)+":"+sub.size()+":"+sub.remove(0)+":"+d+":"+sub);
  ListIterator<Object> li=a.listIterator(1);System.out.println("li:"+li.nextIndex()+":"+li.previousIndex()+":"+li.previous());li.set("Z");li.add("Q");System.out.println("li-write:"+d+":"+li.next()+":"+li.previous());
  Plain p=new Plain();List<Object> v=p;Iterator<Object> pi=v.iterator();System.out.println("plain:"+pi.next()+":"+p.gets+":"+p.sizes+":"+p.generation()+":"+v.add("x")+":"+p.adds+":"+p.generation());
  try{pi.next();System.out.println("bad-cme");}catch(ConcurrentModificationException e){System.out.println("cme:"+p.gets);}
  System.out.println("plain-default:"+v.indexOf("b")+":"+v.lastIndexOf("c")+":"+v.equals(p.data)+":"+(v.hashCode()==p.data.hashCode())+":"+(((Object)p) instanceof RandomAccess));
  Iterator<Object> cleanup=v.iterator();cleanup.next();cleanup.remove();try{cleanup.remove();System.out.println("bad-remove");}catch(IllegalStateException e){System.out.println("cleanup:"+v+":"+p.removes+":"+p.generation());}
 }
}