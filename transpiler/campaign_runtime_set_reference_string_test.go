package transpiler

import "testing"

func TestCampaignRuntimeSetReferenceTextJVM(t *testing.T) {
	campaignCollectionReferenceJDK21(t)
	campaignStringCoreOperationOracle(t, "SetReferenceText", `import java.util.*;
public class SetReferenceText {
 static class Text {
  String text;int calls,ordinary;boolean same,held;final Thread caller=Thread.currentThread();RuntimeException abrupt;Set<Object> mutate;Object remove;
  Text(String text){this.text=text;}
  public String String(){ordinary++;return "impostor";}
  public String toString(){calls++;same=Thread.currentThread()==caller;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;if(mutate!=null)mutate.remove(remove);return text;}
 }
 static String units(String text){StringBuilder out=new StringBuilder();for(int i=0;i<text.length();i++)out.append((int)text.charAt(i)).append(',');return out.toString();}
 public static void main(String[] args){
  Set<Object> empty=new HashSet<>();System.out.println((String.valueOf(empty)=="[]")+":"+(String.valueOf(empty)==String.valueOf(empty)));
  Set<Integer> sorted=new TreeSet<>();sorted.add(10);sorted.add(-1);sorted.add(2);String a=String.valueOf(sorted),b=String.valueOf(sorted);System.out.println(a+":"+(a!=b)+":"+a.equals(b));
  Set<Object> wrappers=new LinkedHashSet<>();wrappers.add(null);wrappers.add(3);wrappers.add(true);System.out.println(String.valueOf(wrappers));
  Set<Object> self=new HashSet<>();self.add(self);System.out.println(String.valueOf(self));
  Text first=new Text("first"),last=new Text(null);Set<Object> callbacks=new LinkedHashSet<>();callbacks.add(first);callbacks.add(last);
  synchronized(first){synchronized(last){System.out.println(String.valueOf(callbacks)+":"+first.calls+":"+last.calls+":"+first.ordinary+":"+last.ordinary+":"+first.same+":"+last.same+":"+first.held+":"+last.held);}}
  Text raw=new Text(new String(new char[]{(char)0xD800,0,(char)0xDFFF}));Set<Object> surrogates=new LinkedHashSet<>();surrogates.add(raw);surrogates.add(Character.valueOf((char)0xDC00));System.out.println(units(String.valueOf(surrogates)));
  Text throwing=new Text("first"),skipped=new Text("last");RuntimeException marker=new IllegalStateException("marker");throwing.abrupt=marker;Set<Object> abrupt=new LinkedHashSet<>();abrupt.add(throwing);abrupt.add(skipped);boolean same=false;try{String.valueOf(abrupt);}catch(RuntimeException got){same=got==marker;}System.out.println(same+":"+throwing.calls+":"+skipped.calls);
  Text mutating=new Text("first"),removed=new Text("last");Set<Object> live=new LinkedHashSet<>();live.add(mutating);live.add(removed);mutating.mutate=live;mutating.remove=removed;boolean cme=false;try{String.valueOf(live);}catch(ConcurrentModificationException got){cme=true;}System.out.println(cme+":"+mutating.calls+":"+removed.calls+":"+live.size());
 }
}
`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
type text struct{value *j.JavaString;calls,ordinary int;same,held bool;caller *j.Execution;abrupt any;mutate *j.Set[any];remove any}
func(*text)JavaDynamicTypeID()j.TypeID{return "settext.Text"}
func(t *text)StringJava2goExecution(*j.Execution)*j.JavaString{t.ordinary++;return lit("impostor")}
func(t *text)DeclaredText(e *j.Execution)*j.JavaString{t.calls++;t.same=e==t.caller;t.held=j.ThreadHoldsLockExecution(e,t);if t.abrupt!=nil{panic(t.abrupt)};if t.mutate!=nil{t.mutate.Remove(t.remove,e)};return t.value}
func lit(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
func host(s *j.JavaString)string{var b strings.Builder;for _,u:=range s.UTF16Copy(){if u>127{panic("non-ASCII observation")};b.WriteByte(byte(u))};return b.String()}
func main(){
 j.RegisterJavaType("settext.Text",j.ObjectTypeID);j.RegisterJavaSourceType("settext.Text");j.RegisterJavaSourceToString("settext.Text","DeclaredText");e:=j.NewExecution()
 empty:=j.NewSet[any]();emptyText:=j.JavaStringValueOfExecution(e,empty);fmt.Printf("%t:%t\n",emptyText==j.JavaStringLiteralUTF16([]uint16{'[',']'}),emptyText==j.JavaStringValueOfExecution(e,empty))
 sorted:=j.NewTreeSet[*j.Integer]();sorted.Add(j.BoxInteger(10),e);sorted.Add(j.BoxInteger(-1),e);sorted.Add(j.BoxInteger(2),e);a,b:=j.JavaStringValueOfExecution(e,sorted),j.JavaStringValueOfExecution(e,sorted);fmt.Printf("%s:%t:%t\n",host(a),a!=b,a.Equals(b))
 wrappers:=j.NewSet[any]();wrappers.Add(nil,e);wrappers.Add(j.BoxInteger(3),e);wrappers.Add(j.BoxBoolean(true),e);fmt.Println(host(j.JavaStringValueOfExecution(e,wrappers)))
 self:=j.NewSet[any]();self.Add(self,e);fmt.Println(host(j.JavaStringValueOfExecution(e,self)))
 first,last:=&text{value:lit("first"),caller:e},&text{caller:e};callbacks:=j.NewSet[any]();callbacks.Add(first,e);callbacks.Add(last,e)
 fg,lg:=j.MonitorEnterExecution(e,first),j.MonitorEnterExecution(e,last);fmt.Printf("%s:%d:%d:%d:%d:%t:%t:%t:%t\n",host(j.JavaStringValueOfExecution(e,callbacks)),first.calls,last.calls,first.ordinary,last.ordinary,first.same,last.same,first.held,last.held);j.MonitorExitExecution(lg);j.MonitorExitExecution(fg)
 raw:=&text{value:j.NewJavaStringUTF16([]uint16{0xD800,0,0xDFFF}),caller:e};surrogates:=j.NewSet[any]();surrogates.Add(raw,e);surrogates.Add(j.BoxCharacter(0xDC00),e);var out strings.Builder;for _,u:=range j.JavaStringValueOfExecution(e,surrogates).UTF16Copy(){fmt.Fprintf(&out,"%d,",u)};fmt.Println(out.String())
 throwing,skipped:=&text{value:lit("first"),caller:e},&text{value:lit("last"),caller:e};marker:=j.NewIllegalStateException("marker");throwing.abrupt=marker;abrupt:=j.NewSet[any]();abrupt.Add(throwing,e);abrupt.Add(skipped,e);same:=false;func(){defer func(){if got:=recover();got!=nil{same=j.JavaReferenceEqual(got,marker)}}();j.JavaStringValueOfExecution(e,abrupt)}();fmt.Printf("%t:%d:%d\n",same,throwing.calls,skipped.calls)
 mutating,removed:=&text{value:lit("first"),caller:e},&text{value:lit("last"),caller:e};live:=j.NewSet[any]();live.Add(mutating,e);live.Add(removed,e);mutating.mutate=live;mutating.remove=removed;cme:=false;func(){defer func(){cme=j.CaughtAs(recover(),"ConcurrentModificationException")}();j.JavaStringValueOfExecution(e,live)}();fmt.Printf("%t:%d:%d:%d\n",cme,mutating.calls,removed.calls,live.Size())
}
`)
}
