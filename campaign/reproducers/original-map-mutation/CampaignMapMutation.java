import java.util.*;
class PutAllKey implements Comparable<PutAllKey>{
 int id;static Map<PutAllKey,String> source;static PutAllKey second;static int mode;static int compares;
 PutAllKey(int id){this.id=id;}
 public int hashCode(){int action=mode;mode=0;if(id==1){if(action==1)source.put(second,"updated");if(action==2)source.put(new PutAllKey(9),"side");}return id;}
 public int compareTo(PutAllKey other){compares++;if(mode==3&&id==2)throw new IllegalStateException("compare failed");return Integer.compare(id,other.id);}
}
public class CampaignMapMutation{
 static String hashCase(int mode){PutAllKey first=new PutAllKey(1);PutAllKey second=new PutAllKey(2);Map<PutAllKey,String> source=new LinkedHashMap<>();source.put(first,"first");source.put(second,"second");PutAllKey.source=source;PutAllKey.second=second;PutAllKey.mode=mode;Map<PutAllKey,String> target=new LinkedHashMap<>();String failure="none";try{target.putAll(source);}catch(ConcurrentModificationException ex){failure="modified";}return target.size()+":"+target.get(first)+":"+target.get(second)+":"+source.size()+":"+failure;}
 public static String run(){String replaced=hashCase(1);String changed=hashCase(2);PutAllKey first=new PutAllKey(1);PutAllKey second=new PutAllKey(2);Map<PutAllKey,String> source=new LinkedHashMap<>();source.put(first,"first");source.put(second,"second");TreeMap<PutAllKey,String> target=new TreeMap<>();PutAllKey.mode=3;String failure="";try{target.putAll(source);}catch(IllegalStateException ex){failure=ex.getMessage();}PutAllKey.mode=0;
 TreeMap<PutAllKey,String> sorted=new TreeMap<>();sorted.put(first,"a");sorted.put(second,"b");PutAllKey.compares=0;TreeMap<PutAllKey,String> copied=new TreeMap<>();copied.putAll(sorted);int calls=PutAllKey.compares;return replaced+"|"+changed+"|"+target.size()+":"+target.get(first)+":"+failure+"|"+copied.size()+":"+calls;
 }} 