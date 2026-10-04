package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignWriterCanonicalStringDispatchJVMParity(t *testing.T) {
	const source = `import java.io.*;
class StringDispatchWriter extends Writer {
 String token;String trace="";
 StringDispatchWriter(String supplied){token=supplied;}
 public void write(String value)throws IOException{trace+="S"+(value==token)+":"+(value==null)+";";if(value!=null)super.write(value);}
 public void write(String value,int off,int len)throws IOException{trace+="R"+(value==token)+":"+(value==null)+":"+off+":"+len+";";if(value!=null)super.write(value,off,len);}
 public void write(char[] data,int off,int len){trace+="C"+Thread.holdsLock(lock)+":";for(int i=0;i<len;i++)trace+=(int)data[off+i]+",";trace+=";";}
 public void flush(){}public void close(){}
}
class WriterCanonicalSequence implements CharSequence {
 public int length(){return 2;}public char charAt(int index){return 'x';}
 public CharSequence subSequence(int start,int end){return new String(new char[]{'\uDFFF'});}
 public String toString(){return new String(new char[]{'\uD800','Q'});}
}
public class CampaignWriterCanonicalStringDispatch {
 public static String run()throws Exception{
  String token=new String(new char[]{'A','\uD800','\uDC00','\uDFFF','Z'});StringDispatchWriter impl=new StringDispatchWriter(token);Writer writer=impl;
  writer.write(token);writer.write(token,1,3);String absent=null;writer.write(absent);writer.write(absent,-7,-9);
  writer.append(token);writer.append(token,1,2);writer.append(new WriterCanonicalSequence());writer.append(new WriterCanonicalSequence(),0,1);writer.append((CharSequence)null);
  return impl.trace;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignWriterCanonicalStringDispatch", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestWriterCanonicalDispatch(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}
