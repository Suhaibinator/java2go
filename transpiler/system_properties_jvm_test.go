package transpiler

import (
	"os"
	"testing"
)

func TestSystemPropertyStatefulJDK21(t *testing.T) {
	source, err := os.ReadFile("../stdjava/testdata/system_properties/SystemPropertyStateOracle.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>property</groupId><artifactId>state</artifactId><version>1</version></project>`, "src/main/java/propertyprobe/SystemPropertyStateOracle.java": string(source)}, "propertyprobe.SystemPropertyStateOracle", "missing:true:true:true\nfirst:true:true:true\nreplace:true:true\nclear:true:true\nempty:true:true\nwhitespace:true\nUTF16:true:true\nprofile:true:21:65.0\nclear-default:true:true\noverride-default:true\ninvalid:0:java.lang.NullPointerException:key can't be null:0\ninvalid:1:java.lang.IllegalArgumentException:key can't be empty:0\ninvalid:2:java.lang.NullPointerException:key can't be null:1\ninvalid:3:java.lang.IllegalArgumentException:key can't be empty:1\ninvalid:4:java.lang.NullPointerException:key can't be null:1\ninvalid:5:java.lang.IllegalArgumentException:key can't be empty:1\ninvalid:6:java.lang.NullPointerException:null:0\ninvalid:7:java.lang.NullPointerException:key can't be null:0\ninvalid:8:java.lang.IllegalArgumentException:key can't be empty:0\ninvalid:9:java.lang.NullPointerException:key can't be null:0\ninvalid:10:java.lang.IllegalArgumentException:key can't be empty:0\nfailed-set-unchanged:true\n")
}

func TestSystemPropertyURLKeysAreOrdinaryPropertiesJDK21(t *testing.T) {
	source, err := os.ReadFile("../stdjava/testdata/system_properties/SystemPropertyURLStateOracle.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>property</groupId><artifactId>url-keys</artifactId><version>1</version></project>`, "src/main/java/propertyprobe/SystemPropertyURLStateOracle.java": string(source)}, "propertyprobe.SystemPropertyURLStateOracle", "set:true:true\nclear:true:true\nset:true:true\nclear:true:true\n")
}

func TestSystemPropertyReferenceIdentityFactsJDK21(t *testing.T) {
	facts, err := selectedSystemPropertyProfile()
	if err != nil {
		t.Fatal(err)
	}
	if facts["JavaVersion"] == "" || facts["RuntimeVersion"] == "" || facts["Vendor"] == "" || facts["VersionDate"] == "" {
		t.Fatal("selected reference release lacks identity facts required by this contract")
	}
	expected := "java.version=" + facts["JavaVersion"] + "\njava.runtime.version=" + facts["RuntimeVersion"] + "\njava.vendor=" + facts["Vendor"] + "\njava.version.date=" + facts["VersionDate"] + "\n"
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>property</groupId><artifactId>identity</artifactId><version>1</version></project>`,
		"src/main/java/propertyprobe/PropertyIdentity.java": `package propertyprobe;public class PropertyIdentity{public static void main(String[] args){System.out.println("java.version="+System.getProperty("java.version"));System.out.println("java.runtime.version="+System.getProperty("java.runtime.version"));System.out.println("java.vendor="+System.getProperty("java.vendor"));System.out.println("java.version.date="+System.getProperty("java.version.date"));}}`,
	}, "propertyprobe.PropertyIdentity", expected)
}
