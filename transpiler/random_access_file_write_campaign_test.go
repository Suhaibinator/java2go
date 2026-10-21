package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestRAFWriteOriginalNativeWriterMainProducer(t *testing.T) {
	resetDiagnostics()
	setStrictMode(false)
	t.Cleanup(func() { resetDiagnostics(); setStrictMode(false) })
	out := renderGoFileFromJava(t, rafWriteOriginalNativeMain)
	for _, operation := range []string{"RandomAccessFileSetLengthExecution(", "RandomAccessFileWriteBytesExecution("} {
		if strings.Count(out, operation) != 1 {
			t.Fatalf("original workflow operation %s absent/duplicated:\n%s", operation, out)
		}
	}
	if len(Diagnostics()) != 0 {
		t.Fatalf("original workflow diagnosed %+v", Diagnostics())
	}
}

const rafWriteOriginalNativeMain = `package writeprobe;
import java.io.RandomAccessFile;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;
public final class NativeWriterMain {
 static void at(String label,FileChannel channel,ByteBuffer src,long position,RandomAccessFile owner){
  try{int writtenBytes=0;writtenBytes+=channel.write(src,position);System.out.println(label+":count="+writtenBytes+":src="+src.position()+"/"+src.limit()+":cursor="+owner.getFilePointer());}
  catch(Throwable failure){System.out.println(label+":"+failure.getClass().getName()+":message="+failure.getMessage()+":cause-null="+(failure.getCause()==null));}
 }
 static void relative(String label,FileChannel channel,ByteBuffer src,RandomAccessFile owner){
  try{int count=channel.write(src);System.out.println(label+":count="+count+":src="+src.position()+"/"+src.limit()+":cursor="+owner.getFilePointer());}
  catch(Throwable failure){System.out.println(label+":"+failure.getClass().getName()+":message="+failure.getMessage()+":cause-null="+(failure.getCause()==null));}
 }
 static void data(RandomAccessFile file)throws Exception {file.seek(0);while(file.getFilePointer()<file.length())System.out.println("byte="+file.read());}
 public static void main(String[] args)throws Exception {
  String path=args[0];try(RandomAccessFile file=new RandomAccessFile(path,"rw")){
   file.setLength(0);file.write(new byte[]{10,11,12,13,14,15,16,17});file.seek(6);FileChannel channel=file.getChannel();
   ByteBuffer root=ByteBuffer.wrap(new byte[]{40,41,42,43,44});root.position(1).limit(4).mark();
   at("positional",channel,root,2,file);root.reset();System.out.println("mark="+root.position());
   ByteBuffer readonly=root.asReadOnlyBuffer();readonly.position(2);at("readonly",channel,readonly,7,file);
   ByteBuffer sliced=root.slice();relative("relative-slice",channel,sliced,file);
   at("empty-max",channel,ByteBuffer.allocate(0),Long.MAX_VALUE,file);
   at("null-negative",channel,null,-1,file);at("negative",channel,ByteBuffer.allocate(1),-1,file);
   relative("relative-null",channel,null,file);data(file);channel.close();
   at("closed-null-negative",channel,null,-1,file);relative("closed-null",channel,null,file);
  }
  try(RandomAccessFile file=new RandomAccessFile(path,"r")){FileChannel channel=file.getChannel();at("readonly-file-null",channel,null,-1,file);relative("readonly-file-null-relative",channel,null,file);at("readonly-file-empty",channel,ByteBuffer.allocate(0),0,file);}
 }
}
`
const rafWriteEvaluationJava = `import java.io.RandomAccessFile;
public class RAFWriteOrder {
 static String trace="", path="", result="";
 static boolean nilBox=false,nilArray=false,empty=false;
 static String name(){trace+="P";return path;}
 static String mode(){trace+="M";return "rw";}
 static Long boxed(){trace+="S";if(nilBox)return null;return 1L;}
 static byte[] bytes(){trace+="A";if(nilArray)return null;if(empty)return new byte[0];return new byte[]{13,-1};}
 static long negative(){trace+="N";return -1L;}
 public static String run(String filePath)throws Exception {
 path=filePath;trace="";result="";nilBox=false;nilArray=false;empty=false;
 try(RandomAccessFile f=new RandomAccessFile(RAFWriteOrder.name(),RAFWriteOrder.mode())){
 f.write(RAFWriteOrder.bytes());f.setLength(RAFWriteOrder.boxed());
 result+="state="+f.getFilePointer()+":"+f.length()+"\n";
 nilBox=true;try{f.setLength(RAFWriteOrder.boxed());}catch(Throwable e){result+="unbox="+e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+"\n";}
 nilArray=true;try{f.write(RAFWriteOrder.bytes());}catch(Throwable e){result+="null="+e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+"\n";}
 f.close();nilArray=false;empty=true;f.write(RAFWriteOrder.bytes());result+="empty=ok\n";
 nilArray=true;try{f.write(RAFWriteOrder.bytes());}catch(Throwable e){result+="closed-null="+e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+"\n";}
 nilArray=false;empty=false;try{f.write(RAFWriteOrder.bytes());}catch(Throwable e){result+="closed-data="+e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+"\n";}
 try{f.setLength(RAFWriteOrder.negative());}catch(Throwable e){result+="closed-negative="+e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+"\n";}
 }
 return trace+"\n"+result;
 }
 public static void main(String[] args)throws Exception{System.out.print(run(args[0]));}
}
`

func TestRAFWriteEvaluationGenerated(t *testing.T) {
	generated := renderGoFileFromJava(t, rafWriteEvaluationJava)
	runGeneratedWithStdjava(t, generated, fmt.Sprintf(`package main
import("testing";"os";"path/filepath";"slices";"github.com/NickyBoy89/java2go/stdjava")
func TestRAFWriteOrdering(t *testing.T){
 path:=filepath.Join(t.TempDir(),"file");got:=Run(stdjava.JavaStringFromHostUTF8(path));want:=stdjava.JavaStringFromHostUTF8(%q)
 if got==nil||!got.Equals(want){t.Fatalf("JDK evaluation mismatch: %%v",got)}
 data,err:=os.ReadFile(path);if err!=nil||!slices.Equal(data,[]byte{13}){t.Fatalf("JDK final bytes %%v/%%v",data,err)}
}`, rafWriteEvaluationJDK))
}

const rafWriteEvaluationJDK = `PMASSAAAAN
state=1:1
unbox=java.lang.NullPointerException:Cannot invoke "java.lang.Long.longValue()" because the return value of "RAFWriteOrder.boxed()" is null:true
null=java.lang.NullPointerException:Cannot read the array length because "b" is null:true
empty=ok
closed-null=java.lang.NullPointerException:Cannot read the array length because "b" is null:true
closed-data=java.io.IOException:Stream Closed:true
closed-negative=java.io.IOException:Stream Closed:true
`
