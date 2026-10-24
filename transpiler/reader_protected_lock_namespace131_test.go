package transpiler

import "testing"

func TestReaderProtectedLockNamespace131(t *testing.T) {
	for _, test := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"publicBridgeCollisionHeld", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Main.java": `package locks;public class Main{static class R extends java.io.Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} public java.lang.Object JavaReaderState(){return this;}boolean same(){return lock==this;}}public static void main(String[]args){R r=new R();System.out.println((r.JavaReaderState()==r)+":"+r.same());}}`}, "true:true\n"},
		{"publicCarrierFieldHeld", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Main.java": `package locks;public class Main{static class R extends java.io.Reader{public java.lang.Object ReaderBase=new java.lang.Object();public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}public static void main(String[]args){R r=new R();System.out.println((r.ReaderBase!=null)+":"+r.same());}}`}, "true:true\n"},
		{"qualifiedExternalReader", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Main.java": `package locks;public class Main{static class R extends other.Reader{Object get(){return lock;}}public static void main(String[]args){R r=new R();System.out.println(r.get()!=r);}}`, "src/main/java/other/Reader.java": `package other;public class Reader{protected Object lock=new Object();}`}, "true\n"},
		{"readSignaturesAndOverloads", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Main.java": `package locks;public class Main{static class R extends java.io.Reader{int calls;public int read(char[]b,int o,int n){calls++;if(n==0)return 0;b[o]='z';return 1;}public void close(){}}static class O extends R{public int read(){calls++;return 9;}String read(char[]b,long o,long n){return "overload";}String read(long n){return "long";}}public static void main(String[]args)throws Exception{R r=new R();char[]b=new char[1];int direct=r.read(b,0,1);O o=new O();System.out.println(r.read()+":"+r.read(b)+":"+direct+":"+r.calls+":"+o.read()+":"+o.read(b,1L,0L)+":"+o.read(1L));}}`}, "122:1:1:3:9:overload:long\n"},
		{"sourceReaderObjectNames", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Main.java": `package locks;class Reader{java.lang.Object lock=new java.lang.Object();}class Object{}public class Main{static class Real extends java.io.Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} java.lang.Object get(){return lock;}}static class Fake extends Reader{java.lang.Object get(){return lock;}}public static void main(String[]args){Real r=new Real();Fake f=new Fake();Object source=new Object();System.out.println((r.get()==r)+":"+(f.get()!=f)+":"+(source!=null));}}`}, "true:true:true\n"},
		{"staticImportAndBinders", map[string]string{"pom.xml": `<project><groupId>reader</groupId><artifactId>extra</artifactId><version>1</version></project>`, "src/main/java/locks/Holder.java": `package locks;public class Holder{public static java.lang.Object lock=new java.lang.Object();}`, "src/main/java/locks/Main.java": `package locks;import static locks.Holder.lock;public class Main{static class R<Reader,Object> extends java.io.Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} java.lang.Object get(){return lock;}}public static void main(String[]args){R<String,Integer> r=new R<String,Integer>();System.out.println((r.get()==r)+":"+(r.get()!=Holder.lock));}}`}, "true:true\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectObservations(t, test.files, "locks.Main", []campaignCompilerProjectObservation{{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want}})
		})
	}
}
