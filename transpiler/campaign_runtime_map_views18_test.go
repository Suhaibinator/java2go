package transpiler

import "testing"

// These JVM-first drivers test runtime adapters directly. Source Iterator
// bridges and canonical Iterable lowering remain independent compiler work.
func TestCampaignRuntimeMapViews18LiveJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "MapViewsLive", `import java.util.*;
public class MapViewsLive {
 static String read(Iterable<String> view){StringBuilder s=new StringBuilder();Iterator<String> i=view.iterator();while(i.hasNext()){if(s.length()>0)s.append('|');s.append(String.valueOf(i.next()));}return s.toString();}
 public static void main(String[] args){
  TreeMap<String,String> map=new TreeMap<>();map.put("a","A");map.put("c","C");
  Set<String> keys=map.keySet();Collection<String> values=map.values();
  map.put("b","B");map.remove("c");map.put("d","D");
  System.out.println("fresh="+read(keys)+":"+read(values));
  Iterator<String> i=values.iterator();System.out.println("peek="+i.hasNext()+":"+i.hasNext()+":"+i.next());
  map.put("b","B2");map.put("d",null);
  System.out.println("replace="+i.next()+":"+String.valueOf(i.next())+":"+i.hasNext());
  System.out.println("again="+read(keys)+":"+read(values));
  map.clear();System.out.println("empty="+read(keys)+":"+read(values));
  map.put("z","Z");System.out.println("refill="+read(keys)+":"+read(values));
 }
}`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
func read(e *j.Execution,view j.JavaIterable)string{var parts []string;i:=j.IterableIteratorExecution(e,view);for j.IteratorHasNextExecution(e,i){parts=append(parts,j.StringValueOfExecution(e,j.IteratorNextExecution(e,i)))};return strings.Join(parts,"|")}
func main(){
 e:=j.NewExecution();m:=j.NewTreeMap[string,string]();m.Put("a","A",e);m.Put("c","C",e)
 keys:=j.MapKeysView(m);values:=j.MapValuesView(m)
 var _ j.Iterable[string]=keys;var _ j.Iterable[string]=values;var _ j.JavaIterable=keys;var _ j.JavaIterable=values
 m.Put("b","B",e);m.Remove("c",e);m.Put("d","D",e)
 fmt.Printf("fresh=%s:%s\n",read(e,keys),read(e,values))
 i:=j.IterableIteratorExecution(e,values);fmt.Printf("peek=%t:%t:%s\n",j.IteratorHasNextExecution(e,i),j.IteratorHasNextExecution(e,i),j.IteratorNextExecution(e,i))
 m.Put("b","B2",e);m.Put("d",j.NullString(),e)
 fmt.Printf("replace=%s:%s:%t\n",j.IteratorNextExecution(e,i),j.StringValueOfExecution(e,j.IteratorNextExecution(e,i)),j.IteratorHasNextExecution(e,i))
 fmt.Printf("again=%s:%s\n",read(e,keys),read(e,values));m.Clear();fmt.Printf("empty=%s:%s\n",read(e,keys),read(e,values));m.Put("z","Z",e);fmt.Printf("refill=%s:%s\n",read(e,keys),read(e,values))
}
`)
}

func TestCampaignRuntimeMapViews18FailureTimingJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "MapViewsTiming", `import java.util.*;
public class MapViewsTiming {
 interface Action{void run();}
 static String outcome(Action a){try{a.run();return "none";}catch(RuntimeException e){return e.getClass().getSimpleName();}}
 static void check(Map<String,String> map,String label){
  Set<String> keys=map.keySet();Collection<String> values=map.values();map.put("a","A");map.put("b","B");
  Iterator<String> k=keys.iterator();Iterator<String> v=values.iterator();
  System.out.println(label+".peek="+k.hasNext()+":"+k.hasNext()+":"+v.hasNext()+":"+v.hasNext());
  System.out.println(label+".first="+outcome(()->k.next())+":"+outcome(()->v.next()));
  map.put("c","C");System.out.println(label+".changed="+k.hasNext()+":"+v.hasNext()+":"+outcome(()->k.next())+":"+outcome(()->v.next()));
  Iterator<String> exhausted=keys.iterator();int count=0;while(exhausted.hasNext()){exhausted.next();count++;}
  System.out.println(label+".exhausted="+(count==map.size())+":"+exhausted.hasNext()+":"+outcome(()->exhausted.next()));
  map.put("d","D");System.out.println(label+".exhaustedChanged="+exhausted.hasNext()+":"+outcome(()->exhausted.next()));
  Iterator<String> removed=values.iterator();map.remove("a");System.out.println(label+".removed="+removed.hasNext()+":"+outcome(()->removed.next()));
  map.clear();Iterator<String> empty=keys.iterator();map.clear();System.out.println(label+".clearEmpty="+empty.hasNext()+":"+outcome(()->empty.next()));
 }
 public static void main(String[] args){check(new TreeMap<>(),"tree");check(new HashMap<>(),"hash");}
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func outcome(action func())(s string){s="none";defer func(){if failure:=recover();failure!=nil{if t,ok:=failure.(j.Throwable);ok{s=t.ThrowableTypeName()}else{panic(failure)}}}();action();return}
func check(e *j.Execution,m *j.Map[string,string],label string){
 keys:=j.MapKeysView(m);values:=j.MapValuesView(m);m.Put("a","A",e);m.Put("b","B",e)
 k:=j.IterableIteratorExecution(e,keys);v:=j.IterableIteratorExecution(e,values)
 fmt.Printf("%s.peek=%t:%t:%t:%t\n",label,j.IteratorHasNextExecution(e,k),j.IteratorHasNextExecution(e,k),j.IteratorHasNextExecution(e,v),j.IteratorHasNextExecution(e,v))
 fmt.Printf("%s.first=%s:%s\n",label,outcome(func(){j.IteratorNextExecution(e,k)}),outcome(func(){j.IteratorNextExecution(e,v)}))
 m.Put("c","C",e);fmt.Printf("%s.changed=%t:%t:%s:%s\n",label,j.IteratorHasNextExecution(e,k),j.IteratorHasNextExecution(e,v),outcome(func(){j.IteratorNextExecution(e,k)}),outcome(func(){j.IteratorNextExecution(e,v)}))
 exhausted:=j.IterableIteratorExecution(e,keys);count:=int32(0);for j.IteratorHasNextExecution(e,exhausted){j.IteratorNextExecution(e,exhausted);count++}
 fmt.Printf("%s.exhausted=%t:%t:%s\n",label,count==m.Size(),j.IteratorHasNextExecution(e,exhausted),outcome(func(){j.IteratorNextExecution(e,exhausted)}))
 m.Put("d","D",e);fmt.Printf("%s.exhaustedChanged=%t:%s\n",label,j.IteratorHasNextExecution(e,exhausted),outcome(func(){j.IteratorNextExecution(e,exhausted)}))
 removed:=j.IterableIteratorExecution(e,values);m.Remove("a",e);fmt.Printf("%s.removed=%t:%s\n",label,j.IteratorHasNextExecution(e,removed),outcome(func(){j.IteratorNextExecution(e,removed)}))
 m.Clear();empty:=j.IterableIteratorExecution(e,keys);m.Clear();fmt.Printf("%s.clearEmpty=%t:%s\n",label,j.IteratorHasNextExecution(e,empty),outcome(func(){j.IteratorNextExecution(e,empty)}))
}
func main(){e:=j.NewExecution();check(e,j.NewTreeMap[string,string](),"tree");check(e,j.NewMap[string,string](),"hash")}
`)
}

func TestCampaignRuntimeMapViews18NullJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "MapViewsNull", `import java.util.*;
public class MapViewsNull {
 interface Action{void run();}
 static String outcome(Action a){try{a.run();return "none";}catch(RuntimeException e){return e.getClass().getSimpleName();}}
 public static void main(String[] args){
  Map<String,String> map=null;Iterable<String> view=null;Iterator<String> iterator=null;
  System.out.println("map="+outcome(()->map.keySet())+":"+outcome(()->map.values()));
  System.out.println("view="+outcome(()->view.iterator()));
  System.out.println("iterator="+outcome(()->iterator.hasNext())+":"+outcome(()->iterator.next()));
  TreeMap<String,String> empty=new TreeMap<>();System.out.println("empty="+empty.keySet().iterator().hasNext()+":"+empty.values().iterator().hasNext());
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func outcome(action func())(s string){s="none";defer func(){if failure:=recover();failure!=nil{if t,ok:=failure.(j.Throwable);ok{s=t.ThrowableTypeName()}else{panic(failure)}}}();action();return}
func main(){
 e:=j.NewExecution();var m *j.Map[string,string];var view j.JavaIterable;var iterator j.JavaIterator
 fmt.Printf("map=%s:%s\n",outcome(func(){j.MapKeysView(m)}),outcome(func(){j.MapValuesView(m)}))
 fmt.Printf("view=%s\n",outcome(func(){j.IterableIteratorExecution(e,view)}))
 fmt.Printf("iterator=%s:%s\n",outcome(func(){j.IteratorHasNextExecution(e,iterator)}),outcome(func(){j.IteratorNextExecution(e,iterator)}))
 empty:=j.NewTreeMap[string,string]();fmt.Printf("empty=%t:%t\n",j.IteratorHasNextExecution(e,j.IterableIteratorExecution(e,j.MapKeysView(empty))),j.IteratorHasNextExecution(e,j.IterableIteratorExecution(e,j.MapValuesView(empty))))
}
`)
}
