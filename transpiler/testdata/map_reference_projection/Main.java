package projection;
import java.util.*;import java.util.concurrent.*;import java.util.function.Function;import java.util.function.Supplier;
public class Main {
 static int reads;static int writes;
 static final class Own implements Map<String,String> {
  final HashMap<String,String> backing=new HashMap<>();
  public int size(){return backing.size();}public boolean isEmpty(){return backing.isEmpty();}
  public boolean containsKey(Object key){return backing.containsKey(key);}public boolean containsValue(Object value){return backing.containsValue(value);}
  public String get(Object key){reads++;return backing.get(key);}public String put(String key,String value){writes++;return backing.put(key,value);}
  public String remove(Object key){return backing.remove(key);}public void putAll(Map<? extends String,? extends String> input){writes+=10;backing.putAll(input);}
  public void clear(){backing.clear();}public Set<String> keySet(){return backing.keySet();}public Collection<String> values(){return backing.values();}public Set<Map.Entry<String,String>> entrySet(){return backing.entrySet();}
 }
 static <T> T read(Map<String,T> map){return map.get("polluted");}
 static Map<String,String> transfer(ConcurrentMap<String,String> value){return value;}
 static String test() {
  ConcurrentMap<String,String> parallel=new ConcurrentHashMap<>();Map<String,String> common=transfer(parallel);
  common.put("key","first");String previous=parallel.putIfAbsent("key","second");
  Map raw=common;raw.put("polluted",Integer.valueOf(7));Object seen=read(common);boolean cast=false;try{String bad=read(common);}catch(ClassCastException expected){cast=true;}
  Object prior=raw.put("polluted","fresh");boolean modified=false;raw.put("polluted",Integer.valueOf(8));try{String bad=common.put("polluted","replacement");}catch(ClassCastException expected){modified=true;}
  Own own=new Own();Map<String,String> source=own;source.put("nil",null);String fallback=source.getOrDefault("nil","fallback");String made=source.putIfAbsent("nil","made");String computed=source.computeIfAbsent("new",key->key+"-value");
  Map<String,String> copied=new HashMap<>();copied.put("copied","copy");source.putAll(copied);
  Function erased=(Object key)->Integer.valueOf(11);Map polluted=new HashMap<String,String>();Object calculated=polluted.computeIfAbsent("function",erased);Object stored=polluted.get("function");
  Supplier<ConcurrentMap<String,String>> factory=ConcurrentHashMap::new;ConcurrentMap<String,String> fresh=factory.get();
  return previous+":"+(seen instanceof Integer)+":"+cast+":"+(prior instanceof Integer)+":"+modified+":"+common.get("polluted")+":"+(fallback==null)+":"+(made==null)+":"+computed+":"+source.get("copied")+":"+(reads>0)+":"+(writes>=12)+":"+(calculated==stored)+":"+(stored instanceof Integer)+":"+fresh.isEmpty();
 }
 public static void main(String[] args){System.out.println(test());}
}