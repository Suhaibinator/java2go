package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignMapPutAllJVMParity(t *testing.T) {
	const source = `import java.util.*;
public class CampaignMapPutAll{
 static String copy(Map<String,String> target,Map<String,String> source){Collection<String> view=target.values();target.putAll(source);target.putAll(target);String values="";for(String value:view){values+=value+",";}source.put("later","source-only");target.put("a","changed");boolean nullRejected=false;try{target.putAll(null);}catch(NullPointerException ex){nullRejected=true;}return target.size()+":"+target.get("a")+":"+(target.get("b")==null)+":"+source.get("a")+":"+target.containsKey("later")+":"+values+":"+nullRejected;}
 public static String run(){Map<String,String> source=new LinkedHashMap<>();source.put("a","one");source.put("b",null);Map<String,String> hash=new LinkedHashMap<>();hash.put("a","old");String first=copy(hash,source);Map<String,String> other=new TreeMap<>();other.put("a","one");other.put("b",null);String second=copy(new TreeMap<String,String>(),other);return first+"|"+second;}
}`
	want := campaignRuntimeJavaOracle(t, "CampaignMapPutAll", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestPutAll(t *testing.T){value:=Run();encoded:=j.JavaStringGetBytes(value,j.UTF_8).Elements;bytes:=make([]byte,len(encoded));for i,b:=range encoded{bytes[i]=byte(b)};if got:=string(bytes);got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}

func TestCampaignMapPutAllMutationJVMParity(t *testing.T) {
	const source = `import java.util.*;
class PutAllKey implements Comparable<PutAllKey>{
 int id;static Map<PutAllKey,String> inputMap;static PutAllKey replacementKey;static int callbackMode;static int compares;
 PutAllKey(int id){this.id=id;}
 public int hashCode(){int action=callbackMode;callbackMode=0;if(id==1){if(action==1)inputMap.put(replacementKey,"updated");if(action==2)inputMap.put(new PutAllKey(9),"side");}return id;}
 public int compareTo(PutAllKey other){compares++;if(callbackMode==3&&id==2)throw new IllegalStateException("compare failed");return Integer.compare(id,other.id);}
}
public class CampaignMapMutation{
 static String hashCase(int mode){PutAllKey first=new PutAllKey(1);PutAllKey second=new PutAllKey(2);Map<PutAllKey,String> source=new LinkedHashMap<>();source.put(first,"first");source.put(second,"second");PutAllKey.inputMap=source;PutAllKey.replacementKey=second;PutAllKey.callbackMode=mode;Map<PutAllKey,String> target=new LinkedHashMap<>();String failure="none";try{target.putAll(source);}catch(ConcurrentModificationException ex){failure="modified";}return target.size()+":"+target.get(first)+":"+target.get(second)+":"+source.size()+":"+failure;}
 public static String run(){String replaced=hashCase(1);String changed=hashCase(2);PutAllKey first=new PutAllKey(1);PutAllKey second=new PutAllKey(2);Map<PutAllKey,String> source=new LinkedHashMap<>();source.put(first,"first");source.put(second,"second");TreeMap<PutAllKey,String> target=new TreeMap<>();PutAllKey.callbackMode=3;String failure="";try{target.putAll(source);}catch(IllegalStateException ex){failure=ex.getMessage();}PutAllKey.callbackMode=0;
 TreeMap<PutAllKey,String> sorted=new TreeMap<>();sorted.put(first,"a");sorted.put(second,"b");PutAllKey.compares=0;TreeMap<PutAllKey,String> copied=new TreeMap<>();copied.putAll(sorted);int calls=PutAllKey.compares;return replaced+"|"+changed+"|"+target.size()+":"+target.get(first)+":"+failure+"|"+copied.size()+":"+calls;
 }} `
	want := campaignRuntimeJavaOracle(t, "CampaignMapMutation", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestPutAllMutation(t *testing.T){value:=Run();encoded:=j.JavaStringGetBytes(value,j.UTF_8).Elements;bytes:=make([]byte,len(encoded));for i,b:=range encoded{bytes[i]=byte(b)};if got:=string(bytes);got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
