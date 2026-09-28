package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignByteArraySubclassAndFilesJVMParity(t *testing.T) {
	const source = `import java.io.*;import java.nio.file.*;
class CountedBytes extends ByteArrayInputStream{
 int closes;
 CountedBytes(byte[] bytes){super(bytes);}
 @Override public void close()throws IOException{closes++;super.close();}
}
class ModifiedBytes extends ByteArrayInputStream{
 ModifiedBytes(byte[] bytes){super(bytes);}
 @Override public int read(){int value=super.read();return value<0?value:value+1;}
 @Override public int read(byte[] bytes,int offset,int length){return super.read(bytes,offset,length);}
}
public class CampaignByteArraySubclass{
 public static String run()throws Exception{
  CountedBytes source=new CountedBytes(new byte[]{10,20,30,40});ByteArrayOutputStream output=new ByteArrayOutputStream();
  try(CountedBytes input=source;ByteArrayOutputStream sink=output){byte[] buffer=new byte[3];int count;while((count=input.read(buffer,0,buffer.length))!=-1){sink.write(buffer,0,count);}}
  File file=File.createTempFile("campaign-byte", ".bin");Path path=file.toPath();Files.write(path,output.toByteArray());byte[] copied=Files.readAllBytes(path);Files.delete(path);
  InputStream modified=new ModifiedBytes(new byte[]{1,2});int first=modified.read();byte[] tail=new byte[1];int last=modified.read(tail,0,1);
  return source.closes+":"+copied.length+":"+copied[0]+":"+copied[3]+":"+first+":"+last+":"+tail[0];
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignByteArraySubclass", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestByteSubclass(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
