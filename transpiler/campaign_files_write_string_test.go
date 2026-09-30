package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignFilesWriteStringJVMParity(t *testing.T) {
	const source = `import java.io.File;import java.nio.file.*;import java.nio.charset.StandardCharsets;
public class CampaignFilesWriteString{public static String run()throws Exception{
 Path path=File.createTempFile("campaign-write", ".txt").toPath();
 StringBuilder text=new StringBuilder();text.append("caf\u00e9");
 Files.writeString(path,text,StandardCharsets.ISO_8859_1);
 byte[] first=Files.newInputStream(path).readAllBytes();
 Files.writeString(path,new StringBuilder("!"),StandardCharsets.ISO_8859_1,StandardOpenOption.APPEND);
 byte[] second=Files.newInputStream(path).readAllBytes();
 Files.writeString(path,new StringBuilder("done"));
 String last=Files.readString(path);
 Files.writeString(path,"plain");String plain=Files.readString(path);Files.delete(path);
 return first.length+":"+first[3]+":"+second.length+":"+second[4]+":"+last+":"+plain;
}}`
	want := campaignRuntimeJavaOracle(t, "CampaignFilesWriteString", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestWriteString(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
