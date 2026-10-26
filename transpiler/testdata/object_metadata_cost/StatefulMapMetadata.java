import java.util.HashMap;
import java.util.Map;

public class StatefulMapMetadata {
 static String trace = "";
 static class Key {
  int id;
  boolean throwing;
  Key(int value) { id=value; }
  public int hashCode() { trace += "H"+id+";"; if(throwing) throw new IllegalStateException("key failure"); return 7; }
  public boolean equals(Object other) { trace += "E"+id+";"; return other instanceof Key && ((Key)other).id==id; }
  public boolean equals(Key other) { throw new IllegalStateException("typed overload selected"); }
  public int hashCodeJava2goExecution(int ignored) { throw new IllegalStateException("foreign member selected"); }
 }
 static class FalseKey extends Key {
  FalseKey(int value) { super(value); }
  public boolean equals(Object other) { trace+="F"+id+";"; return false; }
 }
 public static String run() {
  trace="";
  Map<Object,Integer> map = new HashMap<>();
  String s1=new String(new char[]{'A','\uD800','B'});
  String s2=new String(new char[]{'A','\uD800','B'});
  map.put(s1,11);
  int first=map.get(s2);
  int old=map.put(s2,13);
  map.put(Integer.valueOf(300),17);
  int boxed=map.get(Integer.valueOf(300));
  map.put(Double.valueOf(-0.0),19);
  map.put(Double.valueOf(0.0),23);
  map.put(Double.valueOf(Double.NaN),29);
  int negative=map.get(Double.valueOf(-0.0));
  int positive=map.get(Double.valueOf(0.0));
  int nan=map.get(Double.valueOf(Double.NaN));
  Key one=new Key(1),equal=new Key(1),two=new Key(2);
  map.put(one,31);
  int original=map.get(equal);
  int replaced=map.put(equal,37);
  map.put(two,41);
  int removed=map.remove(new Key(2));
  FalseKey falseKey=new FalseKey(3);
  map.put(falseKey,43);
  int identity=map.get(falseKey);
  boolean falseLookup=map.containsKey(new FalseKey(3));
  boolean directFalse=falseKey.equals((Object)falseKey);
  map.put(null,47);
  int nil=map.get(null);
  int[] a={1,2},b={1,2};map.put(a,53);
  boolean same=map.containsKey(a),different=map.containsKey(b);
  one.throwing=true;
  String failure="";
  try {map.get(one);} catch(IllegalStateException expected) {failure=expected.getMessage();}
  return first+":"+old+":"+boxed+":"+negative+":"+positive+":"+nan+":"+original+":"+replaced+":"+removed+":"+identity+":"+falseLookup+":"+directFalse+":"+nil+":"+same+":"+different+":"+map.size()+":"+s1.equals(s2)+":"+s1.hashCode()+":"+failure+":"+trace;
 }
 public static void main(String[] args) {System.out.print(run());}
}
