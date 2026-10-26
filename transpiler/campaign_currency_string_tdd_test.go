package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCampaignCurrencyStringStrictJVM(t *testing.T) {
	runCurrencyFrozenProject(t, "currency_string_v1", "probe.CurrencyProbe")
}

func TestCampaignCurrencyCompleteCodesStrictJVM(t *testing.T) {
	runCurrencyFrozenProject(t, "currency_codes_v1", "probe.CurrencyCodesProbe")
}

func TestCampaignCurrencySourceForeignBinderGuardsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                              campaignStaticImportPOM26,
		"src/main/java/foreign/Currency.java":  `package foreign;public class Currency{public static Currency getInstance(String code){return new Currency();}public String getCurrencyCode(){return "foreign";}}`,
		"src/main/java/probe/ForeignUse.java":  `package probe;import foreign.Currency;public class ForeignUse{static String run(){Currency value=Currency.getInstance("invalid");return value.getCurrencyCode();}}`,
		"src/main/java/probe/ImportedUse.java": `package probe;import static java.util.Currency.getInstance;public class ImportedUse{static String run(){return getInstance("JPY").getCurrencyCode();}}`,
		"src/main/java/probe/Main.java": `package probe;public class Main{
 static class Currency{static Currency getInstance(String code){return new Currency();}String getCurrencyCode(){return "source";}}
 static class Holder<Currency>{Currency value;Holder(Currency value){this.value=value;}Currency get(){return value;}}
 static java.util.Currency CURRENCY=java.util.Currency.getInstance("USD");
 public static void main(String[] args){Currency value=Currency.getInstance("invalid");Holder<Currency> holder=new Holder<>(value);java.util.Currency nativeValue=java.util.Currency.getInstance("CHF");System.out.println(value.getCurrencyCode()+":"+ForeignUse.run()+":"+ImportedUse.run()+":"+nativeValue.getCurrencyCode()+":"+CURRENCY.getCurrencyCode()+":"+(holder.get()==value));}
}`,
	}, "probe.Main", "source:foreign:JPY:CHF:USD:true\n")
}

func TestCampaignCurrencyNominalOverloadJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.Currency;import java.io.Serializable;public class Main{
 static String pick(Object value){return "Object";}static String pick(Serializable value){return "Serializable";}
 static String trace="";static Currency qualifier(){trace+="q";return null;}static String argument(){trace+="a";return "GBP";}
 public static void main(String[] args){Currency value=Currency.getInstance("GBP");String code=qualifier().getInstance(argument()).getCurrencyCode();System.out.println(trace+":"+code+":"+pick(value)+":"+value.equals(value)+":"+value.equals(null));}
}`,
	}, "probe.Main", "qa:GBP:Serializable:true:false\n")
}

func runCurrencyFrozenProject(t *testing.T, fixture, mainClass string) {
	t.Helper()
	root := filepath.Join("testdata", "campaign", fixture)
	files := map[string]string{}
	for _, name := range []string{"pom.xml", "src/main/java/" + strings.ReplaceAll(mainClass, ".", "/") + ".java"} {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(contents)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, mainClass, []campaignCompilerProjectObservation{
		{name: "repeat-1", stdout: string(expected)}, {name: "repeat-2", stdout: string(expected)}, {name: "repeat-3", stdout: string(expected)},
	})
}

func TestCampaignCurrencyNominalSourceForeignBinderNegativeJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                             campaignStaticImportPOM26,
		"src/main/java/foreign/Currency.java": `package foreign;public class Currency{}`,
		"src/main/java/probe/ForeignUse.java": `package probe;import foreign.Currency;import java.io.Serializable;public class ForeignUse{static String pick(Object value){return "Object";}static String pick(Serializable value){return "Serializable";}static String run(){return pick(new Currency());}}`,
		"src/main/java/probe/Main.java":       `package probe;import java.io.Serializable;public class Main{static class Currency{}static String pick(Object value){return "Object";}static String pick(Serializable value){return "Serializable";}static <Currency> String binder(Currency value){return pick(value);}public static void main(String[] args){System.out.println(pick(new Currency())+":"+ForeignUse.run()+":"+binder(java.util.Currency.getInstance("USD")));}}`,
	}, "probe.Main", "Object:Object:Object\n")
}
