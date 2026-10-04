package transpiler

import (
	"strings"
	"testing"
)

func TestCampaignJavaIntegerParseIntLowering(t *testing.T) {
	out := renderIntrinsicProgram(t, `public class IntegerParseLowering {
        static int decimal(String input){return Integer.parseInt(input);}
        static int radix(String input,int radix){return java.lang.Integer.parseInt(input,radix);}
    }`)
	if count := strings.Count(out, "stdjava.JavaIntegerParseInt("); count != 2 {
		t.Fatalf("canonical helper count=%d, want2:\n%s", count, out)
	}
	if strings.Contains(out, "stdjava.ParseInt(") {
		t.Fatalf("native parse helper remains:\n%s", out)
	}
	own := renderIntrinsicProgram(t, `class Integer {
        static int parseInt(String input){return 7;}
        static int example(){return Integer.parseInt("literal");}
    }`)
	if strings.Contains(own, "stdjava.JavaIntegerParseInt(") {
		t.Fatalf("source Integer captured by builtin lowering:\n%s", own)
	}
}

func TestCampaignJavaIntegerParseIntGeneratedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>integer-parse-reference</artifactId><version>1</version></project>`,
		"src/main/java/parseprobe/Entry.java": `package parseprobe;
import static java.lang.Integer.parseInt;
public class Entry {
    static <T> T identity(T input){return input;}
    static int radix(String input,int radix){return java.lang.Integer.parseInt(input,radix);}
    public static void main(String[] args){
        String copied=new String("+2147483647");
        System.out.println("max="+Integer.parseInt(identity(copied)));
        System.out.println("min="+parseInt("-80000000",16));
        System.out.println("unicode="+radix("１２３",10)+":"+radix("١٢٣",10)+":"+radix("Ｆｆ",16));
        System.out.println("signs="+parseInt("-0")+":"+parseInt("+101",2));
        try{Integer.parseInt("2147483648");System.out.println("overflow=missed");}
        catch(NumberFormatException expected){System.out.println("overflow=NumberFormatException");}
        try{Integer.parseInt((String)null,1);System.out.println("null=missed");}
        catch(NumberFormatException expected){System.out.println("null=NumberFormatException");}
        try{Integer.parseInt("\uD835\uDFD9");System.out.println("supplementary=missed");}
        catch(NumberFormatException expected){System.out.println("supplementary=NumberFormatException");}
    }
}`,
	}, "parseprobe.Entry", "max=2147483647\nmin=-2147483648\nunicode=123:123:255\nsigns=0:5\noverflow=NumberFormatException\nnull=NumberFormatException\nsupplementary=NumberFormatException\n")
}
