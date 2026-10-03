package transpiler

import "testing"

func TestCampaignMapEntrySourceReferenceBridgeJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>entry</groupId><artifactId>bridge</artifactId><version>1</version></project>
`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.Map;
class MutableEntry implements Map.Entry<String,String> {
 final String key; String value; int reads; int writes; boolean held;
 MutableEntry(String key,String value){this.key=key;this.value=value;}
 public String getKey(){return key;}
 public String getValue(){reads++;held=Thread.holdsLock(this);return value;}
 public String setValue(String next){String old=value;value=next;writes++;held=Thread.holdsLock(this);return old;}
}
public class Main {
 static Map.Entry<String,String> pass(Map.Entry<String,String> entry){return entry;}
 public static void main(String[] args){
  MutableEntry source=new MutableEntry("key","before");Map.Entry<String,String> entry=pass(source);Object alias=entry;
  synchronized(source){String key=entry.getKey();Object before=entry.getValue();String old=entry.setValue("after");String after=entry.getValue();
   System.out.println((alias==source)+":"+(alias instanceof Map.Entry)+":"+key+":"+before+":"+old+":"+after+":"+source.reads+":"+source.writes+":"+source.held);
  }
  Map.Entry<String,String> missing=null;boolean caught=false;try{missing.getValue();}catch(NullPointerException expected){caught=true;}System.out.println(caught);
 }
}
`,
	}, "probe.Main", "true:true:key:before:before:after:2:1:true\ntrue\n")
}
