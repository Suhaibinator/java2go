package transpiler

import "testing"

func TestCampaignThrowableCanonicalInitCauseRejection(t *testing.T) {
	campaignStringHostOracle(t, "ThrowableCanonicalInitCause", `public class ThrowableCanonicalInitCause {
 static final ThreadLocal<String> context=new ThreadLocal<>();
 static class Probe extends RuntimeException {
  Throwable primary;String text;RuntimeException abrupt;int calls;boolean held;
  Probe(Throwable target,String value){primary=target;text=value;}
  public String toString(){if(!"ready".equals(context.get()))throw new AssertionError("execution");calls++;held=Thread.holdsLock(primary);if(abrupt!=null)throw abrupt;return text;}
 }
 static void units(String s){for(int i=0;i<s.length();i++)System.out.print((int)s.charAt(i)+",");System.out.println();}
 public static void main(String[] args){
  context.set("ready");
  for(int origin=0;origin<2;origin++)for(String text:new String[]{null,new String(new char[]{'x',(char)0xd800,0,(char)0xdc00})}){
   Exception primary=new Exception((Throwable)null);Probe attempted=new Probe(primary,text);
   try{primary.initCause(attempted);System.out.println("missing");}catch(IllegalStateException failure){System.out.println((failure.getCause()==primary)+":"+(primary.getCause()==null)+":"+attempted.calls+":"+attempted.held);units(failure.getMessage());}
  }
  Exception primary=new Exception((Throwable)null);
  try{primary.initCause(null);System.out.println("missing");}catch(IllegalStateException failure){System.out.println(failure.getCause()==primary);units(failure.getMessage());}
  Probe throwing=new Probe(primary,"unused");RuntimeException exact=new RuntimeException("exact");throwing.abrupt=exact;
  try{primary.initCause(throwing);System.out.println("missing");}catch(RuntimeException failure){System.out.println((failure==exact)+":"+throwing.calls+":"+throwing.held+":"+(primary.getCause()==null));}
  Exception self=new Exception((Throwable)null);try{self.initCause(self);System.out.println("missing");}catch(IllegalStateException failure){System.out.println("initialized-before-self:"+(failure.getCause()==self));}
  Exception legacyPrimary=new Exception((Throwable)null);Probe legacy=new Probe(legacyPrimary,"legacy");
  try{legacyPrimary.initCause(legacy);System.out.println("missing");}catch(IllegalStateException failure){System.out.println((failure.getCause()==legacyPrimary)+":"+legacy.calls+":"+legacy.held);units(failure.getMessage());}
  context.remove();
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type probe struct{j.RuntimeException;primary any;text *j.JavaString;abrupt any;calls int;held bool;execution *j.Execution}
func(*probe)JavaDynamicTypeID()j.TypeID{return "ThrowableCanonicalInitCause$Probe"}
func(p *probe)DeclaredText(e *j.Execution)*j.JavaString{if e!=p.execution{panic("execution")};p.calls++;p.held=j.ThreadHoldsLockExecution(e,p.primary);if p.abrupt!=nil{panic(p.abrupt)};return p.text}
type nativeProbe struct{j.RuntimeException;primary any;calls int;held bool;execution *j.Execution}
func(*nativeProbe)JavaDynamicTypeID()j.TypeID{return "ThrowableCanonicalInitCause$Native"}
func(p *nativeProbe)NativeText(e *j.Execution)string{if e!=p.execution{panic("execution")};p.calls++;p.held=j.ThreadHoldsLockExecution(e,p.primary);return "legacy"}
func units(s *j.JavaString){for _,u:=range s.UTF16Copy(){fmt.Printf("%d,",u)};fmt.Println()}
func main(){e:=j.NewExecution();j.RegisterJavaType("ThrowableCanonicalInitCause$Probe",j.BuiltinThrowableTypeID("RuntimeException"));j.RegisterJavaSourceType("ThrowableCanonicalInitCause$Probe");j.RegisterJavaSourceToString("ThrowableCanonicalInitCause$Probe","DeclaredText")
 for origin:=0;origin<2;origin++{for _,text:=range []*j.JavaString{nil,j.NewJavaStringUTF16([]uint16{'x',0xd800,0,0xdc00})}{primary:=j.NewJavaExceptionCauseExecution(e,nil);if origin==1{primary=j.NewExceptionExecution(e,nil)};attempted:=&probe{RuntimeException:j.NewRuntimeException(),primary:primary,text:text,execution:e};func(){defer func(){if failure:=recover();failure!=nil{if !j.CaughtAs(failure,"IllegalStateException"){panic(failure)};fmt.Printf("%t:%t:%d:%t\n",j.GetCause(failure)==primary,j.GetCause(primary)==nil,attempted.calls,attempted.held);units(j.JavaThrowableMessageDefault(failure))}}();j.ThrowableInitCauseExecution(e,primary,attempted);fmt.Println("missing")}()}}
 primary:=j.NewJavaExceptionCauseExecution(e,nil);func(){defer func(){if failure:=recover();failure!=nil{if !j.CaughtAs(failure,"IllegalStateException"){panic(failure)};fmt.Println(j.GetCause(failure)==primary);units(j.JavaThrowableMessageDefault(failure))}}();j.ThrowableInitCauseExecution(e,primary,nil);fmt.Println("missing")}()
 exact:=j.NewRuntimeException("exact");throwing:=&probe{RuntimeException:j.NewJavaRuntimeExceptionMessage(nil),primary:primary,abrupt:exact,execution:e};func(){defer func(){if failure:=recover();failure!=nil{fmt.Printf("%t:%d:%t:%t\n",failure==exact,throwing.calls,throwing.held,j.GetCause(primary)==nil)}}();j.ThrowableInitCauseExecution(e,primary,throwing);fmt.Println("missing")}()
 self:=j.NewJavaExceptionCauseExecution(e,nil);func(){defer func(){if failure:=recover();failure!=nil{if !j.CaughtAs(failure,"IllegalStateException"){panic(failure)};fmt.Printf("initialized-before-self:%t\n",j.GetCause(failure)==self)}}();j.ThrowableInitCauseExecution(e,self,self);fmt.Println("missing")}()
 j.RegisterJavaType("ThrowableCanonicalInitCause$Native",j.BuiltinThrowableTypeID("RuntimeException"));j.RegisterJavaSourceType("ThrowableCanonicalInitCause$Native");j.RegisterJavaSourceToString("ThrowableCanonicalInitCause$Native","NativeText");legacyPrimary:=j.NewJavaExceptionCauseExecution(e,nil);legacy:=&nativeProbe{RuntimeException:j.NewRuntimeException(),primary:legacyPrimary,execution:e};func(){defer func(){if failure:=recover();failure!=nil{if !j.CaughtAs(failure,"IllegalStateException"){panic(failure)};fmt.Printf("%t:%d:%t\n",j.GetCause(failure)==legacyPrimary,legacy.calls,legacy.held);units(j.JavaThrowableMessageDefault(failure))}}();j.ThrowableInitCauseExecution(e,legacyPrimary,legacy);fmt.Println("missing")}()
}`)
}
