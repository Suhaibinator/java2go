package transpiler

import "testing"

func TestHoistedLocalStaticInterfaceOwnerJVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>owner</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;interface Owner{static void run(){class Cell{int instance=13;static int value=7;}Cell a=new Cell();Cell b=new Cell();a.instance=19;System.out.println(a.instance);System.out.println(b.instance);System.out.println(Cell.value);Cell.value+=2;System.out.println(Cell.value);}}public class Main{public static void main(String[] args){Owner.run();Owner.run();}}
`,
	})
}
