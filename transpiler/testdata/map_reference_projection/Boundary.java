package boundary;
import java.util.*;import java.util.concurrent.*;
public class Main {
 static int calls;
 static final class Own implements Map<String,String>{
  final HashMap<String,String> data=new HashMap<>();
  public int size(){return data.size();}public boolean isEmpty(){return data.isEmpty();}
  public boolean containsKey(Object key){return data.containsKey(key);}public boolean containsValue(Object value){return data.containsValue(value);}
  public String get(Object key){calls++;return data.get(key);}public String put(String key,String value){calls++;return data.put(key,value);}
  public String remove(Object key){return data.remove(key);}public void clear(){data.clear();}
  public Set<String> keySet(){return data.keySet();}public Collection<String> values(){return data.values();}public Set<Map.Entry<String,String>> entrySet(){return data.entrySet();}
  public void putAll(Map<? extends String,? extends String> input){calls+=100;data.putAll(input);}
 }
 static final class Fake {public Object MapEntrySetJava2goExecution(Object ignored){return null;}}
 public static void main(String[] args){
  ConcurrentMap<String,String> concurrent=new ConcurrentHashMap<>();Map<String,String> map=concurrent;
  map.put("first","one");Map raw=map;raw.put("bad",Integer.valueOf(9));Object value=map.get("bad");Object old=map.put("bad","fixed");
  raw.put("bad",Integer.valueOf(8));boolean after=false;try{String ignored=map.put("bad","replacement");}catch(ClassCastException expected){after=true;}
  System.out.println((value instanceof Integer)+":"+(old instanceof Integer)+":"+after+":"+map.get("bad"));
  Own own=new Own();Map<String,String> declared=own;declared.put("nil",null);String original=declared.putIfAbsent("nil","made");String computed=declared.computeIfAbsent("absent",key->key+"-result");
  Map<String,String> input=new HashMap<>();input.put("copy","copied");declared.putAll(input);
  boolean before=false;Map sourceRaw=declared;try{sourceRaw.put("wrong",Integer.valueOf(3));}catch(ClassCastException expected){before=true;}
  System.out.println((original==null)+":"+computed+":"+declared.get("copy")+":"+before+":"+!declared.containsKey("wrong")+":"+(calls>=104));
  Map<String,Integer> primitive=new HashMap<>();Map corrupt=primitive;corrupt.put("wrong",Long.valueOf(3));Object seen=primitive.get("wrong");boolean wrapper=false;try{int ignored=primitive.get("wrong");}catch(ClassCastException expected){wrapper=true;}
  boolean nil=false;try{int ignored=primitive.get("nil");}catch(NullPointerException expected){nil=true;}
  Object nothing=null;Map<String,String> nullMap=(Map<String,String>)nothing;boolean fake=false;try{Map<String,String> rejected=(Map<String,String>)(Object)new Fake();}catch(ClassCastException expected){fake=true;}
  System.out.println((seen instanceof Long)+":"+wrapper+":"+nil+":"+(nullMap==null)+":"+fake);
 }
}