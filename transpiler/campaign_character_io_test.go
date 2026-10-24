package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignReaderSubclassUTF16JVMParity(t *testing.T) {
	const source = `import java.io.*;
class UnitReader extends Reader {
 char[] data=new char[]{'A','\uD83D','\uDE00','Z'};int cursor;int calls;int closes;
 public int read(char[] target,int offset,int length){calls++;if(length==0)return 42;if(cursor==data.length)return -1;int count=Math.min(length,data.length-cursor);for(int i=0;i<count;i++){target[offset+i]=data[cursor++];}return count;}
 public void close(){closes++;}
}
public class CampaignReaderUnits {public static String run()throws Exception{
 UnitReader impl=new UnitReader();Reader reader=impl;int first=reader.read();char[] tail=new char[4];int count=reader.read(tail,1,2);int last=reader.read();int end=reader.read();int empty=reader.read(null,-5,0);reader.close();
 return first+":"+count+":"+(int)tail[1]+":"+(int)tail[2]+":"+last+":"+end+":"+empty+":"+impl.calls+":"+impl.closes;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignReaderUnits", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestReader(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignAnonymousReaderJVMParity(t *testing.T) {
	const source = `import java.io.*;
public class CampaignAnonymousReader {public static String run()throws Exception{
 Reader value=new Reader(){int count;public int read(char[] data,int offset,int length){if(count++==0){data[offset]='Q';return 1;}return -1;}public void close(){throw new AssertionError("closed");}};
 int first=value.read();char[] buffer=new char[1];int end=value.read(buffer);String message="";try{value.close();}catch(AssertionError failure){message=failure.getMessage();}return first+":"+end+":"+message;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignAnonymousReader", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestReader(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignWriterDefaultsUTF16JVMParity(t *testing.T) {
	const source = `import java.io.*;
class UnitWriter extends Writer{
 String trace="";int closes;int flushes;
 public void write(char[] data,int offset,int length){trace+=length+"[";for(int i=0;i<length;i++){trace+=(int)data[offset+i]+",";}trace+="]";}
 public void flush(){flushes++;}public void close(){closes++;}
}
public class CampaignWriterUnits{public static String run()throws Exception{
 UnitWriter impl=new UnitWriter();Writer writer=impl;writer.write(0x1F600);writer.write("A😀Z",1,2);writer.write(new char[]{'Q','R'});writer.append('!');CharSequence absent=null;writer.append(absent);Writer same=writer.append("xyz",1,3);writer.flush();writer.close();return impl.trace+":"+(same==writer)+":"+impl.flushes+":"+impl.closes;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignWriterUnits", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestWriter(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignAppendableSourceDispatchJVMParity(t *testing.T) {
	const source = `import java.io.*;
class Sink implements Appendable,Closeable,Flushable{
 String trace="";
 public Sink append(char value){trace+="c"+(int)value;return this;}
 public Sink append(CharSequence value){trace+="s"+(value==null);return this;}
 public Sink append(CharSequence value,int start,int end){trace+="r"+start+":"+end+":"+(int)value.charAt(start);return this;}
 public void close(){trace+="C";}public void flush(){trace+="F";}
}
class Lookalike{public void close(){}public void flush(){}}
public class CampaignAppendable{public static String run()throws Exception{
 Sink sink=new Sink();Appendable value=sink;value.append('A');value.append(null);value.append("xyz",1,3);
 if(value instanceof Flushable)((Flushable)value).flush();if(value instanceof Closeable)((Closeable)value).close();Object look=new Lookalike();return sink.trace+":"+(look instanceof Closeable)+":"+(look instanceof Flushable);
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignAppendable", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestAppendable(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignWriterMonitorBufferAndVirtualDefaultsJVMParity(t *testing.T) {
	const source = `import java.io.*;
class LockedWriter extends Writer {
 char[] previous;char[] cached;String trace="";boolean fail;
 LockedWriter(Object supplied){super(supplied);}
 public void write(char[] data,int off,int len){trace+=Thread.holdsLock(lock)+":"+data.length+":"+(previous==data)+":"+(cached==data)+":"+len+";";if(cached==null)cached=data;previous=data;synchronized(lock){if(fail)throw new IllegalStateException("callback");}}
 public void write(int value)throws IOException{trace+="I";super.write(value);}
 public void flush(){}public void close(){}
}
class OwnWriter extends Writer{
 String trace="";public void write(char[] data,int off,int len){trace+=(lock==this)+":"+Thread.holdsLock(this);}public void flush(){}public void close(){}
}
class StrangeSequence implements CharSequence {
 public int length(){return 3;}public char charAt(int i){return 'x';}public CharSequence subSequence(int start,int end){return "sub";}public String toString(){return "real";}
}
public class CampaignWriterContracts{public static String run()throws Exception{
 Object lock=new Object();LockedWriter impl=new LockedWriter(lock);Writer writer=impl;writer.write(65);writer.write("bc");writer.write(new String(new char[1025]));writer.write("small");writer.write(new char[]{'d'});
 String bounds="";try{writer.write("short",0,6);}catch(StringIndexOutOfBoundsException expected){bounds=expected.getMessage();}
 try{writer.write("short",2147483647,1);}catch(StringIndexOutOfBoundsException expected){bounds+=":"+expected.getMessage();}
 writer.append(new StrangeSequence());writer.append(new StrangeSequence(),1,2);impl.fail=true;try{writer.write(90);}catch(IllegalStateException expected){}
 OwnWriter own=new OwnWriter();own.write(65);return impl.trace+":"+bounds+":"+Thread.holdsLock(lock)+":"+own.trace;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignWriterContracts", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestWriterContracts(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignAnonymousWriterAndUnrelatedOverloadsJVMParity(t *testing.T) {
	const source = `import java.io.*;
class OverloadedReader extends Reader{
 public int read(int ignored){return 77;}public int read(char[] data,int off,int len){return -1;}public void close(){}
}
class OverloadedWriter extends Writer{
 String trace="";public void write(Object other){trace+="object";}public void write(char[] data,int off,int len){trace+="chars";}public void flush(){}public void close(){}
}
public class CampaignAnonymousWriter{public static String run()throws Exception{
 Writer value=new Writer(){public void write(char[] data,int off,int len){throw new AssertionError("write");}public void flush(){throw new AssertionError("flush");}public void close(){throw new AssertionError("close");}};
 String trace="";try{value.write(65);}catch(AssertionError expected){trace+=expected.getMessage();}try{value.flush();}catch(AssertionError expected){trace+=expected.getMessage();}try{value.close();}catch(AssertionError expected){trace+=expected.getMessage();}
 Object nominal=value;trace+=":"+(nominal instanceof Writer);OverloadedReader reader=new OverloadedReader();OverloadedWriter writer=new OverloadedWriter();Object object=new Object();writer.write(object);return trace+":"+reader.read(1)+":"+writer.trace;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignAnonymousWriter", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestWriter(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}
