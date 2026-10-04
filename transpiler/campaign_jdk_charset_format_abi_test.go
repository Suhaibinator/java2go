package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignCharsetCanonicalNameReferenceABI(t *testing.T) {
	campaignStringCoreOperationOracle(t, "CharsetCanonicalNameReference", `import java.nio.charset.*;
public class CharsetCanonicalNameReference {
 public static void main(String[] args) throws Exception {
  Charset[] sets={StandardCharsets.US_ASCII,StandardCharsets.ISO_8859_1,StandardCharsets.UTF_8,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16};
  String[] names={"US-ASCII","ISO-8859-1","UTF-8","UTF-16BE","UTF-16LE","UTF-16"};
  String[] aliases={"ASCII","LATIN1","utf8","UNICODEBIGUNMARKED","UNICODELITTLEUNMARKED","utf16"};
  Thread[] workers=new Thread[8];boolean[] okay=new boolean[8];
  for(int i=0;i<workers.length;i++){final int index=i;workers[i]=new Thread(()->{boolean good=true;for(int repeat=0;repeat<32;repeat++){for(int c=0;c<sets.length;c++){String one=sets[c].name();good&=one==sets[c].name()&&one==names[c];}}okay[index]=good;});workers[i].start();}
  boolean all=true;for(int i=0;i<workers.length;i++){workers[i].join();all&=okay[i];}System.out.println(all);
  for(int i=0;i<sets.length;i++){
   String first=sets[i].name(),second=sets[i].name();
   System.out.println(first+":"+(first==second)+":"+(first==names[i])+":"+(Charset.forName(aliases[i]).name()==first)+":"+(first!=new String(first)));
  }
  boolean caught=false;try{((Charset)null).name();}catch(NullPointerException expected){caught=true;}
  System.out.println(caught);
 }
}`, `package main
import("fmt";"sync";j "github.com/NickyBoy89/java2go/stdjava")
func literal(s string)*j.JavaString{units:=make([]uint16,len(s));for i:=range s{units[i]=uint16(s[i])};return j.JavaStringLiteralUTF16(units)}
func main(){
 sets:=[]*j.Charset{j.US_ASCII,j.ISO_8859_1,j.UTF_8,j.UTF_16BE,j.UTF_16LE,j.UTF_16}
 names:=[]string{"US-ASCII","ISO-8859-1","UTF-8","UTF-16BE","UTF-16LE","UTF-16"}
 aliases:=[]string{"ASCII","LATIN1","utf8","UNICODEBIGUNMARKED","UNICODELITTLEUNMARKED","utf16"}
 var wg sync.WaitGroup;okay:=make([]bool,8);for i:=range okay{wg.Add(1);go func(index int){defer wg.Done();good:=true;for repeat:=0;repeat<32;repeat++{for c,set:=range sets{one:=j.CharsetJavaName(set);good=good&&one==j.CharsetJavaName(set)&&one==literal(names[c])}};okay[index]=good}(i)};wg.Wait();all:=true;for _,good:=range okay{all=all&&good};fmt.Println(all)
 for i,c:=range sets{first,second:=j.CharsetJavaName(c),j.CharsetJavaName(c);fmt.Printf("%s:%t:%t:%t:%t\n",c.Name(),first==second,first==literal(names[i]),j.CharsetJavaName(j.CharsetForName(aliases[i]))==first,first!=j.CopyJavaString(first))}
 caught:=func()(ok bool){defer func(){ok=j.CaughtAs(recover(),"NullPointerException")}();j.CharsetJavaName(nil);return}();fmt.Println(caught)
}`)
}

func TestCampaignStringFormatUTF16ArgumentsAndAllocation(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringFormatUTF16Arguments", `import java.util.Locale;
public class StringFormatUTF16Arguments {
 static void emit(String s){System.out.print(s.length()+";");for(int i=0;i<s.length();i++)System.out.print((int)s.charAt(i)+",");System.out.println();}
 public static void main(String[] args){
  String value=new String(new char[]{'Q',(char)0xD83D,(char)0xDE03,(char)0xDFFF});
  emit(String.format("A%sB",value));emit(String.format("%.2s",value));emit(String.format("%7.3s",value));emit(String.format("%-7.2s",value));
  emit(String.format(new String(new char[]{(char)0xD800,'%','s',(char)0xDC00}),new String(new char[]{(char)0xDFFF})));
  Object[] fixed={"a","b"};emit(String.format("%2$s:%s:%<s:%1$s",fixed));
  emit(String.format("%%:%n:%s:%b",null,Boolean.FALSE));
  emit(String.format("%s:%d:%b",(Object[])null));
  emit(String.format(Locale.ROOT,"%S|%B|%C|%04x|%+05d", "ß",Boolean.FALSE,'q',26,7));
  emit(String.format(Locale.ROOT,"%h|%H|%c", "z","z",0x1F603));
  emit(String.format(Locale.ROOT,"%x|%o|%X",(byte)-1,(short)-1,-1));
  String first=String.format("%s",value),second=String.format("%s",value);
  System.out.println((first!=value)+":"+(first!=second)+":"+first.equals(second)+":"+(String.format("")==""));
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func lit(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
func emit(s *j.JavaString){fmt.Printf("%d;",s.Length());for _,u:=range s.UTF16Copy(){fmt.Printf("%d,",u)};fmt.Println()}
func main(){e:=j.NewExecution();value:=j.NewJavaStringUTF16([]uint16{'Q',0xD83D,0xDE03,0xDFFF})
 f:=func(s string,values ...any)*j.JavaString{return j.JavaStringFormatExecution(e,lit(s),j.ReferenceArrayLiteral(j.ObjectTypeID,values...))}
 emit(f("A%sB",value));emit(f("%.2s",value));emit(f("%7.3s",value));emit(f("%-7.2s",value))
 emit(j.JavaStringFormatExecution(e,j.NewJavaStringUTF16([]uint16{0xD800,'%','s',0xDC00}),j.ReferenceArrayLiteral(j.ObjectTypeID,j.NewJavaStringUTF16([]uint16{0xDFFF}))))
 fixed:=j.ReferenceArrayLiteral(j.ObjectTypeID,lit("a"),lit("b"));emit(j.JavaStringFormatExecution(e,lit("%2$s:%s:%<s:%1$s"),fixed))
 emit(f("%%:%n:%s:%b",nil,j.BooleanValueOf(false)));emit(j.JavaStringFormatExecution(e,lit("%s:%d:%b"),nil))
 emit(j.JavaStringFormatLocaleExecution(e,j.LocaleROOT,lit("%S|%B|%C|%04x|%+05d"),j.ReferenceArrayLiteral(j.ObjectTypeID,lit("ß"),j.BooleanValueOf(false),j.CharacterValueOf(rune('q')),j.IntegerValueOf(int32(26)),j.IntegerValueOf(int32(7)))))
 emit(j.JavaStringFormatLocaleExecution(e,j.LocaleROOT,lit("%h|%H|%c"),j.ReferenceArrayLiteral(j.ObjectTypeID,lit("z"),lit("z"),j.IntegerValueOf(int32(0x1F603)))))
 emit(j.JavaStringFormatLocaleExecution(e,j.LocaleROOT,lit("%x|%o|%X"),j.ReferenceArrayLiteral(j.ObjectTypeID,j.ByteValueOf(int8(-1)),j.ShortValueOf(int16(-1)),j.IntegerValueOf(int32(-1)))))
 first,second:=f("%s",value),f("%s",value);fmt.Printf("%t:%t:%t:%t\n",first!=value,first!=second,first.Equals(second),f("")==j.JavaStringLiteralUTF16(nil))
}`)
}

func TestCampaignStringFormatExecutionCallbackIdentity(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringFormatExecutionCallbacks", `public class StringFormatExecutionCallbacks {
 static final class Stored {
  final Thread caller=Thread.currentThread();String text;RuntimeException abrupt;int calls,ordinary;boolean same,held,reenter;
  Stored(String text){this.text=text;}
  public String String(){ordinary++;return "impostor";}
  public String toString(){calls++;same=Thread.currentThread()==caller;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;if(reenter)return String.format("[%s]",text);return text;}
 }
 public static void main(String[] args){
  Stored value=new Stored(new String(new char[]{'Q',(char)0xDFFF}));value.reenter=true;
  synchronized(value){
   String out=String.format("%1$s:%<s",value);System.out.println(out.length()+":"+(int)out.charAt(2)+":"+value.calls+":"+value.ordinary+":"+value.same+":"+value.held);
   value.reenter=false;value.text=null;System.out.println(String.format("%s",value)+":"+value.calls);
   String[] formats={"%.1s","%4s","%S"};for(String format:formats){boolean caught=false;try{String.format(format,value);}catch(NullPointerException expected){caught=true;}System.out.println(caught+":"+value.calls);}
   RuntimeException cause=new IllegalArgumentException("cause"),marker=new IllegalStateException("marker",cause);value.abrupt=marker;boolean same=false,sameCause=false;
   try{String.format("%s %s",value,value);}catch(RuntimeException got){same=got==marker;sameCause=got.getCause()==cause;}System.out.println(same+":"+sameCause+":"+value.calls+":"+value.ordinary+":"+value.same+":"+value.held);
  }
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type stored struct{caller *j.Execution;text *j.JavaString;abrupt any;calls,ordinary int;same,held,reenter bool}
func(*stored)JavaDynamicTypeID()j.TypeID{return "format.Stored"}
func(s *stored)StringJava2goExecution(*j.Execution)*j.JavaString{s.ordinary++;return j.JavaStringFromHostUTF8("impostor")}
func(s *stored)DeclaredText(e *j.Execution)*j.JavaString{s.calls++;s.same=e==s.caller;s.held=j.ThreadHoldsLockExecution(e,s);if s.abrupt!=nil{panic(s.abrupt)};if s.reenter{return j.JavaStringFormatExecution(e,j.JavaStringFromHostUTF8("[%s]"),j.ReferenceArrayLiteral(j.ObjectTypeID,s.text))};return s.text}
func main(){j.RegisterJavaType("format.Stored",j.ObjectTypeID);j.RegisterJavaSourceType("format.Stored");j.RegisterJavaSourceToString("format.Stored","DeclaredText")
 e:=j.NewExecution();value:=&stored{caller:e,text:j.NewJavaStringUTF16([]uint16{'Q',0xDFFF}),reenter:true};g:=j.MonitorEnterExecution(e,value);defer j.MonitorExitExecution(g)
 f:=func(format string,values ...any)*j.JavaString{return j.JavaStringFormatExecution(e,j.JavaStringFromHostUTF8(format),j.ReferenceArrayLiteral(j.ObjectTypeID,values...))}
 out:=f("%1$s:%<s",value);fmt.Printf("%d:%d:%d:%d:%t:%t\n",out.Length(),out.CharAt(2),value.calls,value.ordinary,value.same,value.held)
 value.reenter=false;value.text=nil;out=f("%s",value);fmt.Printf("%s:%d\n",stringBytes(out),value.calls)
 for _,format:=range []string{"%.1s","%4s","%S"}{caught:=func()(ok bool){defer func(){ok=j.CaughtAs(recover(),"NullPointerException")}();f(format,value);return}();fmt.Printf("%t:%d\n",caught,value.calls)}
 cause:=j.NewIllegalArgumentException("cause");marker:=j.NewIllegalStateException("marker");j.ThrowableInitCauseExecution(e,marker,cause);value.abrupt=marker;same,sameCause:=false,false
 func(){defer func(){if got:=recover();got!=nil{same=j.JavaReferenceEqual(got,marker);sameCause=j.JavaReferenceEqual(j.GetCause(got),cause)}}();f("%s %s",value,value)}();fmt.Printf("%t:%t:%d:%d:%t:%t\n",same,sameCause,value.calls,value.ordinary,value.same,value.held)
}
func stringBytes(s *j.JavaString)string{b:=j.JavaStringGetBytes(s,j.UTF_8).Elements;out:=make([]byte,len(b));for i,x:=range b{out[i]=byte(x)};return string(out)}`)
}

func TestCampaignStringFormatValidationOrderJDK(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringFormatValidationOrder", `public class StringFormatValidationOrder {
 static final class Counted{int calls;public String toString(){calls++;return "value";}}
 static void outcome(String format,Object[] args){try{String.format(format,args);System.out.println("missing");}catch(RuntimeException e){System.out.println(e.getClass().getSimpleName()+":"+(e instanceof IllegalArgumentException)+":"+e.getMessage());}}
 public static void main(String[] args){
  outcome("%q",new Object[0]);outcome("%s %s",new Object[]{"one"});outcome("%--s",new Object[]{"one"});outcome("%-s",new Object[]{"one"});outcome("%+s",new Object[]{"one"});outcome("%.2d",new Object[]{7});outcome("%0$s",new Object[]{"one"});outcome("%d",new Object[]{"one"});
  Counted value=new Counted();try{String.format("%s%q",value);}catch(RuntimeException e){System.out.println(e.getClass().getSimpleName()+":"+value.calls);}
  try{String.format("%s %s",value);}catch(RuntimeException e){System.out.println(e.getClass().getSimpleName()+":"+value.calls);}
  String rendered=String.format("%s", "one", value);System.out.println(rendered+":"+value.calls);
  boolean caught=false;try{String.format((String)null,new Object[0]);}catch(NullPointerException expected){caught=true;}System.out.println(caught);
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type counted struct{calls int}
func(c *counted)StringJava2goExecution(*j.Execution)*j.JavaString{c.calls++;return j.JavaStringFromHostUTF8("value")}
func main(){e:=j.NewExecution();text:=func(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
 f:=func(s string,values ...any)*j.JavaString{return j.JavaStringFormatExecution(e,text(s),j.ReferenceArrayLiteral(j.ObjectTypeID,values...))}
 outcome:=func(s string,values ...any){func(){defer func(){if x:=recover();x!=nil{v:=x.(j.Throwable);fmt.Printf("%s:%t:%s\n",v.ThrowableTypeName(),j.CaughtAs(x,"IllegalArgumentException"),j.GetMessage(x))}}();f(s,values...);fmt.Println("missing")}()}
 outcome("%q");outcome("%s %s",text("one"));outcome("%--s",text("one"));outcome("%-s",text("one"));outcome("%+s",text("one"));outcome("%.2d",j.IntegerValueOf(int32(7)));outcome("%0$s",text("one"));outcome("%d",text("one"))
 value:=&counted{};for _,s:=range []string{"%s%q","%s %s"}{func(){defer func(){if x:=recover();x!=nil{fmt.Printf("%s:%d\n",x.(j.Throwable).ThrowableTypeName(),value.calls)}}();f(s,value)}()}
 out:=f("%s",text("one"),value);b:=j.JavaStringGetBytes(out,j.UTF_8).Elements;host:=make([]byte,len(b));for i,x:=range b{host[i]=byte(x)};fmt.Printf("%s:%d\n",host,value.calls)
 caught:=func()(ok bool){defer func(){ok=j.CaughtAs(recover(),"NullPointerException")}();j.JavaStringFormatExecution(e,nil,j.ReferenceArrayLiteral(j.ObjectTypeID));return}();fmt.Println(caught)
}`)
}

func TestCampaignCharsetNameCompiledReferenceABI(t *testing.T) {
	const source = `import java.nio.charset.StandardCharsets;
public class CharsetNameCompiledABI {
 static final String ASCII=StandardCharsets.US_ASCII.name(),LATIN=StandardCharsets.ISO_8859_1.name(),UTF8=StandardCharsets.UTF_8.name(),UTF16BE=StandardCharsets.UTF_16BE.name(),UTF16LE=StandardCharsets.UTF_16LE.name(),UTF16=StandardCharsets.UTF_16.name();
 public static String run(){return ASCII+":"+LATIN+":"+UTF8+":"+UTF16BE+":"+UTF16LE+":"+UTF16+":"+(UTF8==StandardCharsets.UTF_8.name())+":"+(UTF8=="UTF-8");}
 public static void main(String[] args){System.out.print(run());}
}`
	want := campaignRuntimeJavaOracle(t, "CharsetNameCompiledABI", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalABI(t *testing.T){ if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatalf("generated result differs from exact JVM stream")}}
`, want))
}

func TestCampaignStringFormatCompiledVarargsABI(t *testing.T) {
	const source = `import java.util.Locale;
import java.util.function.BiFunction;
public class StringFormatCompiledABI {
 final String message;
 StringFormatCompiledABI(String message,Object...args){this.message=String.format(message,args);}
 public static String run(){
  Object[] fixed={"a","b"};Object[] absent={null};
  StringFormatCompiledABI exception=new StringFormatCompiledABI("%2$s:%s:%<s:%1$s",fixed);
  BiFunction<String,Object[],String> formatter=String::format;
  return exception.message+":"+String.format("%b",(Object)absent)+":"+String.format("%b",absent)+":"+String.format("%s",(Object)null)+":"+String.format("%s",(Object[])null)+":"+String.format(Locale.ROOT,"%04x",26)+":"+formatter.apply("%s/%s",fixed);
 }
 public static void main(String[] args){System.out.print(run());}
}`
	want := campaignRuntimeJavaOracle(t, "StringFormatCompiledABI", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalABI(t *testing.T){ if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatalf("generated result differs from exact JVM stream")}}
`, want))
}
