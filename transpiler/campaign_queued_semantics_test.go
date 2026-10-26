package transpiler

import "testing"

func TestCampaignUpdateExpressionBoxing(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>update-boxing</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
public class Main {
 static int n=126;
 static Integer post(){return n++;} static Integer pre(){return ++n;}
 static Integer postDown(){return n--;} static Integer preDown(){return --n;}
 static int accept(Integer value){return value.intValue();}
 public static void main(String[] args){
  System.out.println(post());System.out.println(pre());System.out.println(postDown());System.out.println(preDown());
  Integer local=n++;System.out.println(local);System.out.println(accept(n++));Number wide=++n;System.out.println(wide.intValue());
  Integer boxed=129;Integer original=boxed;Integer previous=boxed++;System.out.println(previous==original);System.out.println(boxed.intValue());
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "126\n128\n128\n126\n126\n127\n129\ntrue\n130\n")
}

func TestCampaignUnbracedBasicFor(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>unbraced-for</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
public class Main {public static void main(String[] args){
 int sum=0;
 for(int i=0;i<3;i++) sum+=i;
 for(int i=0;i<3;i++) if(i==1)continue;else sum+=10;
 outer:for(int i=0;i<3;i++) for(int j=0;j<3;j++) if(j==1)continue outer;else sum++;
 for(int i=0;i<4;i++) if(i==2)break;else sum+=i;
 for(int i=0;i<3;i++);
 System.out.println(sum);
 int[] values={2,3,4};for(int i=0;i<values.length;i++) values[i]++;
 System.out.println(values[0]+values[1]+values[2]);
 int count=0;while(count<2)count++;do count--;while(count>0);
 for(int i=0;i<1;i++)try{count+=2;}finally{count++;}
 System.out.println(count);
}}
`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "27\n12\n3\n")
}

func TestCampaignNarrowingIntegralConstants(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>narrow-constants</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
public class Main {
 static final byte ZERO=(byte)(0xff+1);
 public static void main(String[] args){
 System.out.println((byte)0xed);System.out.println((byte)255);System.out.println((short)65535);
 System.out.println((int)(char)-1);System.out.println((int)0xffffffffL);System.out.println((byte)(128+128));
 final int constant=237;System.out.println((byte)constant);System.out.println((short)-32769);
 long value=0x1000000edL;System.out.println((byte)value);System.out.println((short)value);System.out.println(ZERO);
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "-19\n-1\n-1\n65535\n-1\n0\n-19\n32767\n-19\n237\n0\n")
}
