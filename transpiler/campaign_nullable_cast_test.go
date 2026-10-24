package transpiler

import "testing"

func TestCampaignNullableReferenceCasts(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>nullable-casts</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.io.Writer;
interface Marker {}
class Value implements Marker {}
class Sink extends Writer { String text=""; public void write(char[] data,int offset,int length){ for(int i=0;i<length;i++){text+=data[offset+i];} }public void flush(){}public void close(){} }
public class Main {
 static int calls;static Object missing(){calls++;return null;}
 public static void main(String[] args)throws Exception {
  System.out.println((Object)null==null);
  System.out.println((CharSequence)null==null);
  System.out.println((Marker)null==null);
  System.out.println((Marker)missing()==null);
  Value value=null;Object erased=value;
  System.out.println((CharSequence)erased==null);
  String text=null;Object erasedText=text;
  System.out.println((Marker)erasedText==null);
  System.out.println((String)missing()==null);
  Value present=new Value();Object reference=present;
  System.out.println((Marker)reference==present);
  System.out.println(((CharSequence)(Object)"hello").length());
  Sink sink=new Sink();Writer writer=sink;writer.append((CharSequence)null);System.out.println(sink.text);
  System.out.println(calls);
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "true\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n5\nnull\n2\n")
}
