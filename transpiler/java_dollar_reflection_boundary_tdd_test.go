package transpiler

import "testing"

func TestJavaDollarPrivateReflectionAccessorBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>private-reflection-boundary</artifactId><version>1</version></project>`,
		"src/main/java/p/Method$Bindings.java": `package p;
public class Method$Bindings {
 private int method$(){return 11;}
 private int java2goIdentifier_6a61766132676f24507269766174652f6d6574686f6424(){return 13;}
 public int ordinary(){return 17;}
}`,
		"src/main/java/app/Main.java": `package app;
import p.Method$Bindings;
import java.lang.reflect.Method;
public class Main {public static void main(String[] args) throws Exception {
 Method$Bindings value=new Method$Bindings();
 Method dollar=Method$Bindings.class.getDeclaredMethod("method$");
 Method escaped=Method$Bindings.class.getDeclaredMethod("java2goIdentifier_6a61766132676f24507269766174652f6d6574686f6424");
 dollar.setAccessible(true);
 escaped.setAccessible(true);
 System.out.println(dollar.invoke(value)+":"+escaped.invoke(value)+":"+Method$Bindings.class.getMethod("ordinary").invoke(value));
}}`,
	}, "app.Main", "11:13:17\n")
}

func TestJavaDollarVolatileReflectionAccessorBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>volatile-reflection-boundary</artifactId><version>1</version></project>`,
		"src/main/java/p/Field$Bindings.java": `package p;
public class Field$Bindings {
 public volatile int field$=3;
 public volatile int Java2goIdentifier_4a61766132676f245075626c69632f6669656c6424=5;
 public static volatile int static$=7;
 public int ordinary=19;
}`,
		"src/main/java/app/Main.java": `package app;
import p.Field$Bindings;
import java.lang.reflect.Field;
public class Main {public static void main(String[] args) throws Exception {
 Field$Bindings value=new Field$Bindings();
 Field dollar=Field$Bindings.class.getField("field$");
 Field escaped=Field$Bindings.class.getField("Java2goIdentifier_4a61766132676f245075626c69632f6669656c6424");
 Field statik=Field$Bindings.class.getField("static$");
 Field ordinary=Field$Bindings.class.getField("ordinary");
 System.out.println(dollar.get(value)+":"+escaped.get(value)+":"+statik.get(null)+":"+ordinary.get(value));
 dollar.set(value,11);
 escaped.set(value,13);
 statik.set(null,17);
 ordinary.set(value,23);
 System.out.println(value.field$+":"+value.Java2goIdentifier_4a61766132676f245075626c69632f6669656c6424+":"+Field$Bindings.static$+":"+value.ordinary);
 System.out.println(dollar.get(value)+":"+escaped.get(value)+":"+statik.get(null)+":"+ordinary.get(value));
}}`,
	}, "app.Main", "3:5:7:19\n11:13:17:23\n11:13:17:23\n")
}
