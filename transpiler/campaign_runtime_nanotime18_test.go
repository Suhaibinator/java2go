package transpiler

import "testing"

func TestCampaignRuntimeNanoTime18ElapsedJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>nanotime</groupId><artifactId>elapsed</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static String kind(long value){return "long";}
 static String kind(double value){return "double";}
 public static void main(String[] args)throws Exception {
  long start=System.nanoTime();long previous=start;boolean ordered=true;
  for(int i=0;i<1000;i++){long now=java.lang.System.nanoTime();if(now-previous<0L)ordered=false;previous=now;}
  Thread.sleep(25L);
  long end=System.nanoTime();
  System.out.println(ordered+":"+(end-start>0L)+":"+kind(java.lang.System.nanoTime()));
 }
}`,
	}, "probe.Main", "true:true:long\n")
}

func TestCampaignRuntimeNanoTime18SourceShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>nanotime</groupId><artifactId>shadow</artifactId><version>1</version></project>`,
		"src/main/java/shadow/System.java": `package shadow;
public class System {public static int calls=0;public static long nanoTime(){calls++;return -17L;}}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.System;
public class Main {public static void main(String[] args){
 long first=java.lang.System.nanoTime();long value=System.nanoTime();long last=java.lang.System.nanoTime();
 java.lang.System.out.println((value==-17L)+":"+System.calls+":"+(last-first>=0L));
}}`,
	}, "probe.Main", "true:1:true\n")
}

func TestCampaignRuntimeNanoTime18StaticImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>nanotime</groupId><artifactId>staticimport</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.System.nanoTime;
public class Main {public static void main(String[] args){long start=nanoTime();long end=nanoTime();System.out.println(end-start>=0L);}}`,
	}, "probe.Main", "true\n")
}
