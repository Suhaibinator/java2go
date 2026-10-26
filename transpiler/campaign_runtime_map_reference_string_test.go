package transpiler

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCampaignRuntimeMapReferenceTextJVM(t *testing.T) {
	campaignCollectionReferenceJDK21(t)
	campaignStringCoreOperationOracle(t, "MapReferenceText", `import java.util.*;
public class MapReferenceText {
 static class Text {
  String text;int calls,ordinary; boolean same,held;final Thread caller=Thread.currentThread();RuntimeException abrupt;
  Map<Object,Object> mutate;Object replacement;
  Text(String text){this.text=text;}
  public String String(){ordinary++;return "impostor";}
  public String toString(){calls++;same=Thread.currentThread()==caller;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;if(mutate!=null)mutate.put(this,replacement);return text;}
 }
 static String units(String text){StringBuilder out=new StringBuilder();for(int i=0;i<text.length();i++)out.append((int)text.charAt(i)).append(',');return out.toString();}
 public static void main(String[] args){
  Map<Object,Object> empty=new LinkedHashMap<>();System.out.println((String.valueOf(empty)=="{}")+":"+(String.valueOf(empty)==String.valueOf(empty)));
  Map<Object,Object> plain=new TreeMap<>();plain.put(1,"a");plain.put(2,null);String first=String.valueOf(plain),second=String.valueOf(plain);System.out.println(first+":"+(first!=second)+":"+first.equals(second));
  Map<Object,Object> self=new LinkedHashMap<>();self.put(self,self);System.out.println(String.valueOf(self));
  Text key=new Text("key"),value=new Text(null);Map<Object,Object> callbacks=new LinkedHashMap<>();callbacks.put(key,value);
  synchronized(key){synchronized(value){System.out.println(String.valueOf(callbacks)+":"+key.calls+":"+value.calls+":"+key.ordinary+":"+value.ordinary+":"+key.same+":"+value.same+":"+key.held+":"+value.held);}}
  Text raw=new Text(new String(new char[]{(char)0xD800,0,(char)0xDFFF}));Map<Object,Object> surrogates=new LinkedHashMap<>();surrogates.put("s",raw);System.out.println(units(String.valueOf(surrogates)));
  Text mutating=new Text("k");Map<Object,Object> live=new LinkedHashMap<>();Object before="old",after="new";live.put(mutating,before);mutating.mutate=live;mutating.replacement=after;System.out.println(String.valueOf(live)+":"+(live.get(mutating)==after)+":"+mutating.calls);
  Text throwing=new Text("key"),skipped=new Text("value");RuntimeException marker=new IllegalStateException("marker");throwing.abrupt=marker;Map<Object,Object> abrupt=new LinkedHashMap<>();abrupt.put(throwing,skipped);boolean same=false;try{String.valueOf(abrupt);}catch(RuntimeException got){same=got==marker;}System.out.println(same+":"+throwing.calls+":"+skipped.calls);
 }
}
`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
type text struct{value *j.JavaString;calls,ordinary int;same,held bool;caller *j.Execution;abrupt any;mutate *j.Map[any,any];replacement any}
func(*text)JavaDynamicTypeID()j.TypeID{return "maptext.Text"}
func(t *text)StringJava2goExecution(*j.Execution)*j.JavaString{t.ordinary++;return lit("impostor")}
func(t *text)DeclaredText(e *j.Execution)*j.JavaString{t.calls++;t.same=e==t.caller;t.held=j.ThreadHoldsLockExecution(e,t);if t.abrupt!=nil{panic(t.abrupt)};if t.mutate!=nil{t.mutate.Put(t,t.replacement,e)};return t.value}
func lit(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
func host(s *j.JavaString)string{var b strings.Builder;for _,u:=range s.UTF16Copy(){if u>127{panic("non-ASCII observation")};b.WriteByte(byte(u))};return b.String()}
func main(){
 j.RegisterJavaType("maptext.Text",j.ObjectTypeID);j.RegisterJavaSourceType("maptext.Text");j.RegisterJavaSourceToString("maptext.Text","DeclaredText");e:=j.NewExecution()
 empty:=j.NewMap[any,any]();emptyText:=j.JavaStringValueOfExecution(e,empty);fmt.Printf("%t:%t\n",emptyText==j.JavaStringLiteralUTF16([]uint16{'{','}'}),emptyText==j.JavaStringValueOfExecution(e,empty))
 plain:=j.NewTreeMap[any,any]();plain.Put(j.BoxInteger(1),lit("a"),e);plain.Put(j.BoxInteger(2),nil,e);first,second:=j.JavaStringValueOfExecution(e,plain),j.JavaStringValueOfExecution(e,plain);fmt.Printf("%s:%t:%t\n",host(first),first!=second,first.Equals(second))
 self:=j.NewMap[any,any]();self.Put(self,self,e);fmt.Println(host(j.JavaStringValueOfExecution(e,self)))
 key,value:=&text{value:lit("key"),caller:e},&text{caller:e};callbacks:=j.NewMap[any,any]();callbacks.Put(key,value,e)
 kg,vg:=j.MonitorEnterExecution(e,key),j.MonitorEnterExecution(e,value);fmt.Printf("%s:%d:%d:%d:%d:%t:%t:%t:%t\n",host(j.JavaStringValueOfExecution(e,callbacks)),key.calls,value.calls,key.ordinary,value.ordinary,key.same,value.same,key.held,value.held);j.MonitorExitExecution(vg);j.MonitorExitExecution(kg)
 raw:=&text{value:j.NewJavaStringUTF16([]uint16{0xD800,0,0xDFFF}),caller:e};surrogates:=j.NewMap[any,any]();surrogates.Put(lit("s"),raw,e);var out strings.Builder;for _,u:=range j.JavaStringValueOfExecution(e,surrogates).UTF16Copy(){fmt.Fprintf(&out,"%d,",u)};fmt.Println(out.String())
 mutating:=&text{value:lit("k"),caller:e};live:=j.NewMap[any,any]();before,after:=lit("old"),lit("new");live.Put(mutating,before,e);mutating.mutate=live;mutating.replacement=after;fmt.Printf("%s:%t:%d\n",host(j.JavaStringValueOfExecution(e,live)),j.JavaReferenceEqual(live.Get(mutating,e),after),mutating.calls)
 throwing,skipped:=&text{value:lit("key"),caller:e},&text{value:lit("value"),caller:e};marker:=j.NewIllegalStateException("marker");throwing.abrupt=marker;abrupt:=j.NewMap[any,any]();abrupt.Put(throwing,skipped,e);same:=false;func(){defer func(){if got:=recover();got!=nil{same=j.JavaReferenceEqual(got,marker)}}();j.JavaStringValueOfExecution(e,abrupt)}();fmt.Printf("%t:%d:%d\n",same,throwing.calls,skipped.calls)
}
`)
}

// Select the installed JDK 21 for these Java 21 operation oracles when the
// caller has not selected JAVA_HOME. macOS's unqualified java may use a newer JDK.
func campaignCollectionReferenceJDK21(t *testing.T) {
	t.Helper()
	if os.Getenv("JAVA_HOME") != "" || runtime.GOOS != "darwin" {
		return
	}
	result := campaignStringBoundedRun(t, "JDK 21 discovery", ".", "/usr/libexec/java_home", 60*time.Second, "-v", "21")
	campaignStringRequireSuccess(t, "JDK 21 discovery", result)
	home := strings.TrimSpace(string(result.stdout))
	if home == "" {
		t.Fatal("installed JDK 21 resolver returned an empty JAVA_HOME")
	}
	t.Setenv("JAVA_HOME", home)
}
