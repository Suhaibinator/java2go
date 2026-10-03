package transpiler

import "testing"

func TestCIPrintWriterCanonicalPathnameContract(t *testing.T) {
	const source = `import java.io.*;import java.nio.file.*;
public class CIWriterPathname {
 public static String run() throws Exception {
  File file=File.createTempFile("ci-path-\u00e9-\ud83d\ude00", ".txt");
  String path=file.getPath();file.delete();
  PrintWriter writer=new PrintWriter(path);writer.println("caf\u00e9\ud83d\ude00");writer.close();
  String read=Files.readString(file.toPath());boolean created=file.exists();
  PrintWriter truncate=new PrintWriter(path);truncate.print("last");truncate.close();
  boolean truncated=Files.readString(file.toPath()).equals("last");
  String raw=path+"\ud800";PrintWriter odd=new PrintWriter(raw);odd.print("odd");odd.close();
  boolean replaced=new File(path+"?").exists();boolean rawDeleted=new File(raw).delete();
  String missing="none";try{new PrintWriter(path+"/missing/leaf");}catch(Exception ex){missing=ex.getClass().getName();}
  String nul="none";try{new PrintWriter("bad\u0000path");}catch(Exception ex){nul=ex.getClass().getName()+":"+ex.getMessage();}
  String absent=null;boolean nullRejected=false;try{new PrintWriter(absent);}catch(NullPointerException ex){nullRejected=true;}
  boolean deleted=file.delete();
  return read.length()+":"+(int)read.charAt(4)+":"+(int)read.charAt(5)+":"+(int)read.charAt(6)+":"
   +created+":"+truncated+":"+replaced+":"+rawDeleted+":"+missing+":"+nul+":"+nullRejected+":"+deleted+":"+file.exists();
 }
}`
	verifyCanonicalStringStreamOracle(t, "CIWriterPathname", source)
}
