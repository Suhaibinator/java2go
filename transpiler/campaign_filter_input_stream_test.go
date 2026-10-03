package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignFilterInputStreamJVMParity(t *testing.T) {
	const source = `import java.io.*; import java.nio.charset.StandardCharsets;
class CountingInput extends FilterInputStream {
 int bytes; int closed;
 CountingInput(InputStream source){super(source);}
 @Override public int read() throws IOException {int value=super.read();if(value>=0)bytes++;return value;}
 @Override public int read(byte[] buffer,int offset,int length)throws IOException{int count=in.read(buffer,offset,length);if(count>0)bytes+=count;return count;}
 @Override public void close()throws IOException{if(closed==0){try{super.close();}finally{closed++;}}}
}
public class CampaignFilterInputStream {
 public static String run() throws Exception {
  CountingInput counted=new CountingInput(new ByteArrayInputStream("caf\u00e9\nsecond".getBytes(StandardCharsets.UTF_8)));
  InputStream base=counted; int head=base.read();
  String first; String second;
  try(CountingInput tracked=counted;BufferedReader reader=new BufferedReader(new InputStreamReader(tracked,StandardCharsets.UTF_8))){first=reader.readLine();second=reader.readLine();}
  return head+":"+first+":"+second+":"+counted.bytes+":"+counted.closed;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignFilterInputStream", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestFilterInput(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(want)), want))
}

func TestCampaignFilterInputStreamOverloadsJVMParity(t *testing.T) {
	const source = `import java.io.*; import java.nio.charset.StandardCharsets;
class OverloadedFilter extends FilterInputStream {
 int calls;
 OverloadedFilter(InputStream input){super(input);}
 public int read(int ignored){return 99;}
 public int read(int a,int b,int c){return 88;}
 @Override public int read()throws IOException{calls++;return super.read();}
 public void close(int ignored){}
 @Override public void close()throws IOException{super.close();}
}
public class CampaignFilterOverloads{
 public static String run()throws Exception{
  OverloadedFilter filter=new OverloadedFilter(new ByteArrayInputStream(new byte[]{65,66,67}));
  InputStream input=filter;
  int first=input.read();byte[] rest=new byte[2];int count=input.read(rest);
  input.close();
  return first+":"+count+":"+rest[0]+":"+rest[1]+":"+filter.calls;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignFilterOverloads", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestFilterOverloads(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(want)), want))
}

func TestCampaignFilterInputStreamEmptyReadOverrideJVMParity(t *testing.T) {
	const source = `import java.io.*;
class EmptyFilter extends FilterInputStream{
 EmptyFilter(){super(new ByteArrayInputStream(new byte[0]));}
 @Override public int read(byte[] bytes,int offset,int length){return 42;}
}
public class CampaignFilterEmpty{public static String run()throws Exception{InputStream input=new EmptyFilter();return input.read(new byte[0],0,0)+":"+input.read(null,-1,0);}}`
	want := campaignRuntimeJavaOracle(t, "CampaignFilterEmpty", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestEmptyRead(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(want)), want))
}
