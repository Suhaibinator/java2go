package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCampaignGenericMethodReferenceResult(t *testing.T) {
	root := "../campaign/reproducers/generic-method-reference-result"
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".java" && rel != "pom.xml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerProjectOracle(t, files, "q.Bridge", "abc\n")
}
func TestCampaignGenericMethodReferenceResultContracts(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>methodref</artifactId><version>1</version></project>`,
		"src/main/java/p/RefResult.java": `package p;
import java.util.function.Function;
import java.util.function.Supplier;
interface UnboundRef {String apply(RefResult receiver,String value);}
interface IntResult {int get();}
public class RefResult {
 static int calls;
 public <T>T id(T value){calls++;return value;}
 @SuppressWarnings("unchecked") public <T>T polluted(T value){calls++;return (T)(Object)Integer.valueOf(7);}
 public <T>T empty(){calls++;return null;}
 public static void main(String[]args){
  RefResult receiver=new RefResult();
  Function<String,String> bound=receiver::id;
  UnboundRef unbound=RefResult::id;
  System.out.println(bound.apply("x"));System.out.println(unbound.apply(receiver,"y"));
  Function<String,Object> wide=receiver::polluted;
  System.out.println(wide.apply("ignored"));
  Function<String,String> narrow=receiver::polluted;
  boolean cast=false;try{String ignored=narrow.apply("ignored");}catch(ClassCastException expected){cast=true;}
  System.out.println(cast);
  Supplier<String> nullable=receiver::empty;
  System.out.println(nullable.get()==null);
  IntResult primitive=receiver::empty;
  boolean unbox=false;try{int ignored=primitive.get();}catch(NullPointerException expected){unbox=true;}
  System.out.println(unbox);
  RefResult absent=null;boolean eager=false;
  try{Function<String,String> ignored=absent::id;}catch(NullPointerException expected){eager=true;}
  System.out.println(eager);System.out.println(calls);
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "p.RefResult", "x\ny\n7\ntrue\ntrue\ntrue\ntrue\n6\n")
}

func TestCampaignGenericMethodReferenceWideningAndDiscard(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>methodref-widening</artifactId><version>1</version></project>`,
		"src/main/java/p/WidenResult.java": `package p;
class ParentValue {public int number=9;}
class ChildValue extends ParentValue {}
interface Widening {ParentValue apply(ChildValue value);}
public class WidenResult {
 static int calls;
 public <T>T id(T value){calls++;return value;}
 public <T>T empty(){calls++;return null;}
 public static void main(String[]args){
  WidenResult receiver=new WidenResult();
  Widening widen=receiver::id;
  System.out.println(widen.apply(new ChildValue()).number);
  Runnable discard=receiver::empty;discard.run();
  System.out.println(calls);
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "p.WidenResult", "9\n2\n")
}

func TestCampaignGenericMethodReferenceShadowedResultBinder(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>methodref-shadow</artifactId><version>1</version></project>`,
		"src/main/java/p/ShadowResult.java": `package p;
interface IntResult {int get();}
class GenericOwner<T> {public <T>T empty(){return null;}}
public class ShadowResult {
 public static void main(String[]args){
  GenericOwner<String> receiver=new GenericOwner<String>();
  IntResult value=receiver::empty;
  boolean rejected=false;try{int ignored=value.get();}catch(NullPointerException expected){rejected=true;}
  System.out.println(rejected);
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "p.ShadowResult", "true\n")
}
