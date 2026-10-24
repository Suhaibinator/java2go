package transpiler

import "testing"

// This is a JVM-derived runtime ABI test, not a compiler parity claim. The Go
// driver represents the intended pointer-returning Java execution methods.
func TestCampaignSourceJavaStringRegisteredReferenceJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "SourceReferenceText", `public class SourceReferenceText {
 static final class Stored {
  String value;final Thread caller=Thread.currentThread();int calls,ordinary;boolean held,same;RuntimeException abrupt;
  Stored(String value){this.value=value;}
  public String String(){ordinary++;return "impostor";}
  public String toString(){calls++;held=Thread.holdsLock(this);same=Thread.currentThread()==caller;if(abrupt!=null)throw abrupt;return value;}
 }
 public static void main(String[] args){
  String payload=new String(new char[]{(char)0xD800,0,(char)0xDC00});Stored value=new Stored(payload);
  synchronized(value){
   String first=String.valueOf((Object)value),second=String.valueOf((Object)value);
   System.out.println((first==payload)+":"+(first==second)+":"+first.length()+":"+(int)first.charAt(0)+":"+(int)first.charAt(1)+":"+(int)first.charAt(2)+":"+value.calls+":"+value.ordinary+":"+value.held+":"+value.same);
   value.value=null;System.out.println((String.valueOf((Object)value)==null)+":"+value.calls+":"+value.ordinary);
   RuntimeException marker=new IllegalStateException("marker");value.abrupt=marker;boolean same=false;
   try{String.valueOf((Object)value);}catch(RuntimeException got){same=got==marker;}
   System.out.println(same+":"+value.calls+":"+value.ordinary+":"+value.held+":"+value.same);
  }
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type stored struct{ value *j.JavaString; caller *j.Execution; calls,ordinary int;held,same bool; abrupt any }
func(*stored)JavaDynamicTypeID()j.TypeID{return "reference.Stored"}
func(s *stored)StringJava2goExecution(*j.Execution)*j.JavaString{s.ordinary++;return j.JavaStringLiteralUTF16([]uint16{'i','m','p','o','s','t','o','r'})}
func(s *stored)DeclaredText(e *j.Execution)*j.JavaString{s.calls++;s.held=j.ThreadHoldsLockExecution(e,s);s.same=e==s.caller;if s.abrupt!=nil{panic(s.abrupt)};return s.value}
func main(){
 j.RegisterJavaType("reference.Stored",j.ObjectTypeID);j.RegisterJavaSourceType("reference.Stored");j.RegisterJavaSourceToString("reference.Stored","DeclaredText")
 e:=j.NewExecution();payload:=j.NewJavaStringUTF16([]uint16{0xD800,0,0xDC00});value:=&stored{value:payload,caller:e}
 guard:=j.MonitorEnterExecution(e,value);defer j.MonitorExitExecution(guard)
 first,second:=j.JavaStringValueOfExecution(e,value),j.JavaStringValueOfExecution(e,value)
 fmt.Printf("%t:%t:%d:%d:%d:%d:%d:%d:%t:%t\n",first==payload,first==second,first.Length(),first.CharAt(0),first.CharAt(1),first.CharAt(2),value.calls,value.ordinary,value.held,value.same)
 value.value=nil;fmt.Printf("%t:%d:%d\n",j.JavaStringValueOfExecution(e,value)==nil,value.calls,value.ordinary)
 marker:=j.NewIllegalStateException("marker");value.abrupt=marker;same:=false
 func(){defer func(){if got:=recover();got!=nil{same=j.JavaReferenceEqual(got,marker)}}();j.JavaStringValueOfExecution(e,value)}()
 fmt.Printf("%t:%d:%d:%t:%t\n",same,value.calls,value.ordinary,value.held,value.same)
}`)
}

func TestCampaignSourceJavaStringInheritedReentryJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "SourceInheritedText", `public class SourceInheritedText {
 static final class Inner {
  String value;int calls,ordinary;Inner(String value){this.value=value;}
  public String String(){ordinary++;return "inner-impostor";}
  public String toString(){calls++;return value;}
 }
 static class Base<T> {
  Inner inner;int calls,ordinary;boolean held,same;final Thread caller=Thread.currentThread();
  Base(Inner inner){this.inner=inner;}
  public String String(){ordinary++;return "base-impostor";}
  public String toString(){calls++;held=Thread.holdsLock(this);same=Thread.currentThread()==caller;return String.valueOf((Object)inner);}
 }
 static final class Child extends Base<String>{Child(Inner inner){super(inner);}}
 public static void main(String[] args){
  String payload=new String(new char[]{'Q',(char)0xDFFF});Inner inner=new Inner(payload);Child child=new Child(inner);Base<String> base=child;
  synchronized(child){
   String first=String.valueOf((Object)base);String second=String.valueOf((Object)child);
   System.out.println((first==payload)+":"+(first==second)+":"+child.calls+":"+inner.calls+":"+child.ordinary+":"+inner.ordinary+":"+child.held+":"+child.same);
   inner.value=null;System.out.println((String.valueOf((Object)base)==null)+":"+child.calls+":"+inner.calls+":"+child.held);
  }
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type inner struct{value *j.JavaString;calls,ordinary int}
func(*inner)JavaDynamicTypeID()j.TypeID{return "reference.Inner"}
func(i *inner)StringJava2goExecution(*j.Execution)*j.JavaString{i.ordinary++;return j.JavaStringLiteralUTF16([]uint16{'i'})}
func(i *inner)DeclaredText(*j.Execution)*j.JavaString{i.calls++;return i.value}
type base struct{*j.ObjectInfo;inner *inner;caller *j.Execution;calls,ordinary int;held,same bool}
func(b *base)StringJava2goExecution(*j.Execution)*j.JavaString{b.ordinary++;return j.JavaStringLiteralUTF16([]uint16{'b'})}
func(b *base)DeclaredText(e *j.Execution)*j.JavaString{b.calls++;b.held=j.ThreadHoldsLockExecution(e,b);b.same=e==b.caller;return j.JavaStringValueOfExecution(e,b.inner)}
type child struct{base}
func main(){
 for _,id:=range []j.TypeID{"reference.Inner","reference.Base"}{j.RegisterJavaType(id,j.ObjectTypeID);j.RegisterJavaSourceType(id);j.RegisterJavaSourceToString(id,"DeclaredText")}
 j.RegisterJavaType("reference.Child","reference.Base");j.RegisterJavaSourceType("reference.Child")
 e:=j.NewExecution();payload:=j.NewJavaStringUTF16([]uint16{'Q',0xDFFF});inside:=&inner{value:payload};value:=&child{base:base{inner:inside,caller:e}}
 value.ObjectInfo=j.NewObjectInfo("reference.Child",func(id j.TypeID)any{if id=="reference.Base"{return &value.base};return value})
 baseView:=&value.base;guard:=j.MonitorEnterExecution(e,value);defer j.MonitorExitExecution(guard)
 first,second:=j.JavaStringValueOfExecution(e,baseView),j.JavaStringValueOfExecution(e,value)
 fmt.Printf("%t:%t:%d:%d:%d:%d:%t:%t\n",first==payload,first==second,value.calls,inside.calls,value.ordinary,inside.ordinary,value.held,value.same)
 inside.value=nil;fmt.Printf("%t:%d:%d:%t\n",j.JavaStringValueOfExecution(e,baseView)==nil,value.calls,inside.calls,value.held)
}`)
}
