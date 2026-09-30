package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignOutputStreamNarrowWriteJVMParity(t *testing.T) {
	const source = `import java.io.*;import java.nio.file.*;
public class CampaignOutputStreamNarrow{public static String run()throws Exception{
 byte escape=(byte)'%';short accent=233;char letter='A';
 ByteArrayOutputStream out=new ByteArrayOutputStream();out.write(escape);out.write(accent);out.write(letter);out.write(511);
 byte[] bytes=out.toByteArray();File file=File.createTempFile("narrow-write",".bin");
 try(FileOutputStream stream=new FileOutputStream(file)){stream.write(escape);stream.write(accent);}
 byte[] disk;try(FileInputStream stream=new FileInputStream(file)){disk=stream.readAllBytes();}Files.delete(file.toPath());
 return bytes.length+":"+bytes[0]+":"+bytes[1]+":"+bytes[2]+":"+bytes[3]+":"+disk.length+":"+disk[0]+":"+disk[1];
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignOutputStreamNarrow", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestNarrow(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
