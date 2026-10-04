package transpiler

import "testing"

func TestCampaignExceptionReferenceEquality(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>exception-identity</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
class BaseFailure extends Exception{public BaseFailure initCause(Throwable cause){super.initCause(cause);return this;}}
class DerivedFailure extends BaseFailure{}
public class Main{public static void main(String[] args){
 DerivedFailure actual=new DerivedFailure();BaseFailure sourceBase=actual;
 Throwable returned=sourceBase.initCause(null);Exception exception=actual;Throwable throwable=actual;
 System.out.println((returned==actual)+":"+(exception==returned)+":"+(throwable==sourceBase));
 Throwable missing=null;Exception other=new Exception();
 System.out.println((missing==null)+":"+(null==missing)+":"+(missing!=actual)+":"+(returned!=other));
 Object broad=actual;System.out.println((broad==returned)+":"+(sourceBase==exception));
}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "true:true:true\ntrue:true:true:true\ntrue:true\n")
}
