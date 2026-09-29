import java.util.*;
import java.util.concurrent.*;
import java.util.stream.*;

public class CanonicalStringContainers {
  static String units(String s) {
    if (s == null) return "null";
    StringJoiner out = new StringJoiner(",");
    for (int i = 0; i < s.length(); i++) out.add(Integer.toString(s.charAt(i)));
    return out.toString();
  }
  public static void main(String[] args) throws Exception {
    String a = new String(new char[]{'x', '\uD800', 0, '\uDC00'}), b = new String(a);
    Map<String,String> map = new HashMap<>();
    map.put(a,a); map.put(b,b);
    System.out.println("map=" + map.size()+","+(map.keySet().iterator().next()==a)+","+(map.get(a)==b)+","+(map.get("missing")==null));
    Set<String> set = new HashSet<>(); set.add(a); set.add(b);
    IdentityHashMap<String,String> identity = new IdentityHashMap<>(); identity.put(a,a); identity.put(b,b);
    System.out.println("setIdentity="+set.size()+","+identity.size()+","+(a==b));
    String[] values = new String[2]; Object[] erased = values;
    System.out.println("null="+(erased[0]==null)+","+(((String)erased[0])==null));
    erased[0]=a; erased[1]=null;
    System.out.println("views="+(values[0]==a)+","+(erased[1]==null)+","+(a instanceof CharSequence));
    try { erased[1]=new Object(); } catch (ArrayStoreException e) { System.out.println("store=ArrayStoreException"); }
    try { String bad=(String)new Object(); } catch (ClassCastException e) { System.out.println("cast=ClassCastException"); }
    String pair=new String(new char[]{'\uD83D','\uDE00'}), bmp=new String(new char[]{'\uE000'});
    System.out.println("compare="+Comparator.<String>naturalOrder().compare(pair,bmp));
    try { ((Comparable)a).compareTo(new Object()); } catch (ClassCastException e) { System.out.println("compareCast=ClassCastException"); }
    try { Comparator.<String>naturalOrder().compare(a,null); } catch (NullPointerException e) { System.out.println("compareNull=NullPointerException"); }
    CharSequence cs=a;
    System.out.println("sequence="+cs.length()+","+(int)cs.charAt(1)+","+(int)cs.charAt(3));
    StringBuilder builder=new StringBuilder().append(a).insert(1,b);
    String first=builder.toString(), second=builder.toString(); builder.append('z');
    System.out.println("builder="+units(first)+","+(first!=second)+","+first.equals(second));
    System.out.println("join="+units(String.join(new String(new char[]{'\uDFFF'}),a,null,b)));
    System.out.println("stream="+units(Stream.of(a,null,b).collect(Collectors.joining("|","[","]"))));
    System.out.println("region="+a.regionMatches(false,1,b,1,3)+","+a.regionMatches(false,0,b,0,-1));
    ConcurrentHashMap<String,String> concurrent=new ConcurrentHashMap<>(); concurrent.put(a,a);
    Thread[] threads=new Thread[8];
    for(int i=0;i<threads.length;i++) { threads[i]=new Thread(()->{for(int j=0;j<128;j++){String copy=new String(a); if(copy.hashCode()!=a.hashCode())throw new AssertionError(); concurrent.put(copy,copy);}}); threads[i].start(); }
    for(Thread thread:threads)thread.join();
    System.out.println("concurrent="+concurrent.size()+","+(concurrent.keySet().iterator().next()==a)+","+concurrent.get(b).equals(a));
  }
}
