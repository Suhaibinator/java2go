package transpiler

import (
	"strings"
	"testing"
)

// Every exception message is observed as UTF16 units before stdout encoding.
// The JDK oracle supplies the expectations for strict/race generated execution.
func TestCI111SQLStringBoundaryJDK21(t *testing.T) {
	const source = `public class Main {
 static String units(String value){if(value==null)return "null";String result=""+value.length();for(int i=0;i<value.length();i++)result=result+","+(int)value.charAt(i);return result;}
 static String parse(int kind,String text){try{return kind==0?java.sql.Date.valueOf(text).toString():kind==1?java.sql.Time.valueOf(text).toString():java.sql.Timestamp.valueOf(text).toString();}catch(IllegalArgumentException failure){return failure.getClass().getSimpleName()+":"+units(failure.getMessage());}}
 public static String run(){
  java.util.TimeZone original=java.util.TimeZone.getDefault();java.util.TimeZone.setDefault(java.util.TimeZone.getTimeZone("UTC"));
  java.sql.Date date=java.sql.Date.valueOf("2023-2-29");java.sql.Time time=java.sql.Time.valueOf("25:61:61");java.sql.Timestamp stamp=java.sql.Timestamp.valueOf(" 2024-01-01 01:02:03.00100 ");
  String dateText=date.toString();String timeText=time.toString();String stampText=stamp.toString();
  String result=dateText+"|"+timeText+"|"+stampText+"|"+(dateText!=date.toString())+":"+(timeText!=time.toString())+":"+(stampText!=stamp.toString())+":"+dateText.equals(date.toString());
  Object erased=stamp;result=result+"|"+String.valueOf(erased)+"|"+erased;
  result=result+"|"+parse(0,null)+"|"+parse(1,null)+"|"+parse(2,null);
  result=result+"|"+parse(0,"\uFF12\uFF10\uFF12\uFF14-\uFF10\uFF11-\uFF10\uFF12")+"|"+parse(1,"1:+2:03");
  result=result+"|"+parse(1,"1::2")+"|"+parse(1,"1:\uD800:03")+"|"+parse(1,"1:2:2147483648");
  result=result+"|"+parse(2,"2024-01-01 :02:03")+"|"+parse(2,"2024-01-01 01:02:x.")+"|"+parse(2,"2024-01-01 01:02:x.1234567890")+"|"+parse(2,"2024-01-01 01:02:03.\uDC00");
  java.sql.Date missingDate=null;java.sql.Time missingTime=null;java.sql.Timestamp missingStamp=null;int caught=0;
  try{missingDate.toString();}catch(NullPointerException expected){caught++;}try{missingTime.toString();}catch(NullPointerException expected){caught++;}try{missingStamp.toString();}catch(NullPointerException expected){caught++;}
  result=result+"|"+caught;java.util.TimeZone.setDefault(java.util.TimeZone.getTimeZone("GMT+05:45"));result=result+"|"+date.toString()+"|"+time.toString()+"|"+stamp.toString();
  java.util.TimeZone.setDefault(original);return result;
 }
 public static void main(String[] args){System.out.println(run());}
}`
	want := campaignRuntimeJavaOracle(t, "Main", source) + "\n"
	runCampaignThrowableStrictSingleFileOracle(t, source, want)
}

func TestCI111SQLStringIntrinsicSourceShadows(t *testing.T) {
	for _, owner := range []string{"Date", "Time", "Timestamp"} {
		t.Run(owner, func(t *testing.T) {
			source := "class " + owner + " { static String valueOf(String text){return text;} public String toString(){return \"source\";} } class Owner { static String run(){return " + owner + ".valueOf(\"source\");} }"
			generated := renderGoFileFromJava(t, source)
			if strings.Contains(generated, "stdjava.SQL") {
				t.Fatalf("source declaration selected a SQL intrinsic:\n%s", generated)
			}
			// The canonical owner helper assumes its caller has resolved lexical
			// declarations. Exercise those real callers with the same binder,
			// then a method binder bounded by a source receiver declaration.
			binderSource := "class Receiver { String valueOf(String text){return text;} } class Owner<" + owner + "> { " + owner + " field; " + owner + " same(" + owner + " value){return value;} static <" + owner + " extends Receiver> String bound(" + owner + " value){return value.valueOf(\"source\");} }"
			bound := renderGoFileFromJava(t, binderSource)
			if strings.Contains(bound, "stdjava.SQL") || !strings.Contains(bound, ".valueOfJava2goExecution(") {
				t.Fatalf("binder field/argument/source receiver acquired SQL lowering:\n%s", bound)
			}
		})
	}
}
