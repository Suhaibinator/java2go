package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// Named codec tests compare unchanged Java bodies with translated Go and exact
// UTF16 results. They deliberately cover supported encodings beyond type shape.
func codecNamedReferenceOracle(t *testing.T, class, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, class, source)
	t.Logf("JDK21 named codec oracle %s=%q", class, want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("slices";"testing";"unicode/utf16";j "github.com/NickyBoy89/java2go/stdjava")
func TestNamedCodecOracle(t *testing.T){var got *j.JavaString=Run();if got==nil{t.Fatal("Run returned null")};want:=utf16.Encode([]rune(%q));if units:=got.UTF16Copy();!slices.Equal(units,want){t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x",want,units)}}`, want))
}

func TestCampaignCodecNamedEncodeJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedEncode", `public class CodecNamedEncode {
 public static String run() throws Exception {
  String[] names={"UTF-8","US-ASCII","ISO-8859-1","UTF-16BE","UTF-16LE","UTF-16"};
  char[][] values={{},{0,'A',(char)0xe9},{(char)0xd83d,(char)0xde00},{'x',(char)0xd83d,'y',(char)0xde00},{(char)0xd800,(char)0xd801,(char)0xdc00,(char)0xdc01},{(char)0xfffd},{(char)0xfeff,'A'}};
  StringBuilder out=new StringBuilder();
  for(String name:names){for(char[] units:values){String value=new String(units);byte[] a=value.getBytes(name),b=value.getBytes(name);out.append(a!=b).append(':');for(byte x:a){out.append((int)x).append(',');}out.append('/');for(int i=0;i<value.length();i++){out.append((int)value.charAt(i)).append(',');}out.append('|');}}
  return out.toString();
 }
}`)
}

func TestCampaignCodecNamedDecodeJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedDecode", `public class CodecNamedDecode {
 public static String run() throws Exception {
  String[] names={"UTF-8","US-ASCII","ISO-8859-1","UTF-16BE","UTF-16LE","UTF-16"};
  byte[][] values={{},{65,0,66},{-61,-87},{-16,-97,-104,-128},{-19,-96,-128},{-30,-126,65},{-30,-126},{-40,0,0,65},{0,-40,65,0},{-1,-2,65,0},{-2,-1,0,65},{-1},{-40,0,0},{-40,0,-40,0,-36,0}};
  StringBuilder out=new StringBuilder();
  for(String name:names){for(byte[] bytes:values){String a=new String(bytes,name),b=new String(bytes,name);out.append(a!=b).append(':').append(a.equals(b)).append(':');for(int i=0;i<a.length();i++){out.append((int)a.charAt(i)).append(',');}out.append('|');}}
  byte[] bytes={65,66};String value=new String(bytes,"UTF8");bytes[0]=90;byte[] exposed=value.getBytes("UTF8");exposed[1]=90;out.append((int)value.charAt(0)).append(':').append((int)value.charAt(1)).append(':').append((int)value.getBytes("UTF8")[1]);
  return out.toString();
 }
}`)
}

func TestCampaignCodecNamedAliasesJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedAliases", `import java.nio.charset.*;
public class CodecNamedAliases {
 public static String run() throws Exception {
  String[] names={"UTF-8","UTF8","unicode-1-1-utf-8","UTF-16","UTF_16","utf16","unicode","UnicodeBig","UTF-16BE","UTF_16BE","ISO-10646-UCS-2","X-UTF-16BE","UnicodeBigUnmarked","UTF-16LE","UTF_16LE","X-UTF-16LE","UnicodeLittleUnmarked","ISO-8859-1","iso-ir-100","ISO_8859-1","latin1","l1","IBM819","cp819","csISOLatin1","819","IBM-819","ISO8859_1","ISO_8859-1:1987","ISO_8859_1","8859_1","ISO8859-1","US-ASCII","iso-ir-6","ANSI_X3.4-1986","ISO_646.irv:1991","ASCII","ISO646-US","us","IBM367","cp367","csASCII","646","iso_646.irv:1983","ANSI_X3.4-1968","ascii7"};
  Charset[] expected={StandardCharsets.UTF_8,StandardCharsets.UTF_8,StandardCharsets.UTF_8,StandardCharsets.UTF_16,StandardCharsets.UTF_16,StandardCharsets.UTF_16,StandardCharsets.UTF_16,StandardCharsets.UTF_16,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16LE,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.ISO_8859_1,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII,StandardCharsets.US_ASCII};
  StringBuilder out=new StringBuilder();
  for(int i=0;i<names.length;i++){String name=names[i];out.append(Charset.forName(name)==expected[i]).append(':').append(new String("A".getBytes(name),name).equals("A")).append('|');}
  out.append(Charset.forName("uTf8")==StandardCharsets.UTF_8);
  return out.toString();
 }
}`)
}

func TestCampaignCodecNamedContractsJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedContracts", `import java.io.*;import java.nio.charset.*;
public class CodecNamedContracts {
 static String units(String value){if(value==null)return "null";StringBuilder out=new StringBuilder();for(int i=0;i<value.length();i++)out.append((int)value.charAt(i)).append(',');return out.toString();}
 static String encode(String name){try{"A".getBytes(name);return "ok";}catch(UnsupportedEncodingException e){return "UEE:"+(e.getMessage()==name)+":"+units(e.getMessage());}catch(NullPointerException e){return "NPE";}}
 static String decode(String name){try{new String(new byte[0],name);return "ok";}catch(UnsupportedEncodingException e){return "UEE:"+(e.getMessage()==name)+":"+units(e.getMessage());}catch(NullPointerException e){return "NPE";}}
 static String lookup(String name){try{Charset.forName(name);return "ok";}catch(IllegalCharsetNameException e){return "ICN";}catch(UnsupportedCharsetException e){return "UCS";}catch(IllegalArgumentException e){return "IAE:"+units(e.getMessage());}}
 public static String run(){String[] names={null,"","UTF_8","x-java2go-no-such-encoding-467923","bad name","!bad","x+none:part.2",new String(new char[]{'A',(char)0xd800})};StringBuilder out=new StringBuilder();for(String name:names)out.append(lookup(name)).append('/').append(encode(name)).append('/').append(decode(name)).append('|');return out.toString();}
}`)
}

func TestCampaignCodecNamedEvaluationJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedEvaluation", `import java.io.*;
public class CodecNamedEvaluation {
 static String trace=""; static byte[] stored;
 static String receiver(){trace+="R";return null;}
 static String name(int mode){trace+="N";if(mode==1)throw new IllegalStateException();if(mode==2)return "x-java2go-no-such-encoding-467923";if(mode==3)return null;return "UTF8";}
 static byte[] input(boolean absent){trace+="B";return absent?null:new byte[]{65,66};}
 static int offset(){trace+="I";return -1;}
 static int count(){trace+="C";return 2;}
 static String mutatingName(){trace+="M";stored=null;return "UTF8";}
 public static String run() throws Exception {
  StringBuilder out=new StringBuilder();
  try{receiver().getBytes(name(1));}catch(IllegalStateException e){trace+="E";}out.append(trace).append('|');trace="";
  try{receiver().getBytes(name(2));}catch(NullPointerException e){trace+="P";}out.append(trace).append('|');trace="";
  try{new String(input(true),name(2));}catch(UnsupportedEncodingException e){trace+="U";}out.append(trace).append('|');trace="";
  try{new String(input(true),name(0));}catch(NullPointerException e){trace+="P";}out.append(trace).append('|');trace="";
  try{new String(input(false),offset(),count(),name(2));}catch(UnsupportedEncodingException e){trace+="U";}out.append(trace).append('|');trace="";
  try{new String(input(false),offset(),count(),name(0));}catch(StringIndexOutOfBoundsException e){trace+="X";}out.append(trace).append('|');trace="";
  try{new String(input(true),name(3));}catch(NullPointerException e){trace+="P";}out.append(trace).append('|');trace="";
  stored=new byte[]{65};String value=new String(stored,mutatingName());out.append(trace).append(':').append((int)value.charAt(0)).append(':').append(stored==null);
  return out.toString();
 }
}`)
}

func TestCampaignCodecNamedRangeJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecNamedRange", `import java.io.*;import java.nio.charset.*;
public class CodecNamedRange {
 static String units(String value){StringBuilder out=new StringBuilder();for(int i=0;i<value.length();i++)out.append((int)value.charAt(i)).append(',');return out.toString();}
 public static String run() throws Exception {
  byte[] value={90,65,-61,-87,66,90};StringBuilder out=new StringBuilder();
  out.append(units(new String(value,1,4,"UTF8"))).append('|').append(units(new String(value,1,4,StandardCharsets.UTF_8))).append('|').append(units(new String(value,1,4))).append('|');
  int[][] bounds={{-1,1},{1,-1},{5,2},{2147483647,2147483647}};
  for(int[] bound:bounds){try{new String(value,bound[0],bound[1],"UTF8");out.append("missing");}catch(StringIndexOutOfBoundsException e){out.append("SIOOB:").append(e.getMessage());}out.append('|');}
  out.append(new String(value,2,0,"UTF8").length()).append(':').append(new String(value,6,0,"UTF8").length());
  return out.toString();
 }
}`)
}

func TestCampaignCodecSourceCharsetShadowJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecSourceCharsetShadow", `class Charset {static String forName(String name){return name;}}
public class CodecSourceCharsetShadow {public static String run(){String value=new String("source");return (Charset.forName(value)==value)+":"+Charset.forName("UTF_8");}}`)
}

func TestCampaignCodecCharsetForNameReferenceAST(t *testing.T) {
	const source = `import java.nio.charset.Charset;
public class CodecCharsetLookup {public static Object run(){return Charset.forName("UTF8");}}`
	generated := renderGoFileFromJava(t, source)
	if !strings.Contains(generated, "stdjava.CharsetForNameReference(") || strings.Contains(generated, "stdjava.CharsetForName(") {
		t.Fatalf("canonical charset lookup must retain JavaString: %s", generated)
	}
}

// Installed nonmandatory codecs are explicit runtime boundaries, not falsely
// modeled Java unsupported-charsets. This JVM-only witness is availability
// evidence; the separate runtime blocker assertion is not Java parity.
func TestCampaignCodecJDKInstalledProviderWitness(t *testing.T) {
	const source = `import java.nio.charset.Charset;
public class CodecProviderWitness {public static String run(){return Charset.forName("Cp1252").name()+"/"+Charset.forName("UTF-32").name()+"/"+Charset.forName("Shift_JIS").name()+"/"+Charset.forName("UnicodeLittle").name();}}`
	got := campaignRuntimeJavaOracle(t, "CodecProviderWitness", source)
	t.Logf("JDK21 installed provider witness=%q", got)
	if got != "windows-1252/UTF-32/Shift_JIS/x-UTF-16LE-BOM" {
		t.Fatalf("pinned JDK21 provider availability changed: %q", got)
	}
}

// Proves the source-derived availability registry against the installed target
// JVM. This metadata check does not claim the unimplemented codecs can encode.
func TestCampaignCodecJDKBundledRegistryWitness(t *testing.T) {
	const source = `import java.nio.charset.*;import java.util.*;
public class CodecRegistryWitness {public static String run(){TreeSet<String> names=new TreeSet<>();for(Charset codec:Charset.availableCharsets().values()){names.add(codec.name().toLowerCase(Locale.ROOT));for(String alias:codec.aliases())names.add(alias.toLowerCase(Locale.ROOT));}return String.join("\n",names);}}`
	got := campaignRuntimeJavaOracle(t, "CodecRegistryWitness", source)
	const expected = "037\n1006\n1025\n1026\n1046\n1047\n1089\n1097\n1098\n1112\n1122\n1123\n1124\n1129\n1140\n1141\n1142\n1143\n1144\n1145\n1146\n1147\n1148\n1149\n1166\n1364\n1381\n1383\n273\n277\n278\n280\n284\n285\n290\n29626c\n297\n300\n33722\n420\n424\n437\n500\n5601\n646\n737\n775\n813\n819\n833\n834\n838\n850\n852\n855\n856\n857\n858\n860\n861\n862\n863\n864\n865\n866\n868\n869\n870\n871\n874\n875\n8859_1\n8859_13\n8859_15\n8859_2\n8859_3\n8859_4\n8859_5\n8859_6\n8859_7\n8859_8\n8859_9\n912\n913\n914\n915\n916\n918\n920\n921\n922\n923\n930\n932\n933\n935\n937\n939\n942\n942c\n943\n943c\n948\n949\n949c\n950\n964\n970\nansi-1251\nansi_x3.4-1968\nansi_x3.4-1986\narabic\nascii\nascii7\nasmo-708\nbig5\nbig5-hkscs\nbig5-hkscs-2001\nbig5-hkscs:unicode3.0\nbig5_hkscs\nbig5_hkscs_2001\nbig5_solaris\nbig5hk\nbig5hk-2001\nbig5hkscs\nbig5hkscs-2001\nccsid00858\nccsid01140\nccsid01141\nccsid01142\nccsid01143\nccsid01144\nccsid01145\nccsid01146\nccsid01147\nccsid01148\nccsid01149\ncesu-8\ncesu8\ncns11643\ncp-ar\ncp-gr\ncp-is\ncp00858\ncp01140\ncp01141\ncp01142\ncp01143\ncp01144\ncp01145\ncp01146\ncp01147\ncp01148\ncp01149\ncp037\ncp1006\ncp1025\ncp1026\ncp1046\ncp1047\ncp1089\ncp1097\ncp1098\ncp1112\ncp1122\ncp1123\ncp1124\ncp1129\ncp1140\ncp1141\ncp1142\ncp1143\ncp1144\ncp1145\ncp1146\ncp1147\ncp1148\ncp1149\ncp1166\ncp1250\ncp1251\ncp1252\ncp1253\ncp1254\ncp1255\ncp1256\ncp1257\ncp1258\ncp1364\ncp1381\ncp1383\ncp273\ncp277\ncp278\ncp280\ncp284\ncp285\ncp290\ncp29626c\ncp297\ncp300\ncp33722\ncp367\ncp420\ncp424\ncp437\ncp500\ncp50220\ncp50221\ncp5346\ncp5347\ncp5348\ncp5349\ncp5350\ncp5353\ncp737\ncp775\ncp813\ncp819\ncp833\ncp834\ncp838\ncp850\ncp852\ncp855\ncp856\ncp857\ncp858\ncp860\ncp861\ncp862\ncp863\ncp864\ncp865\ncp866\ncp868\ncp869\ncp870\ncp871\ncp874\ncp875\ncp912\ncp913\ncp914\ncp915\ncp916\ncp918\ncp920\ncp921\ncp922\ncp923\ncp930\ncp932\ncp933\ncp935\ncp936\ncp937\ncp939\ncp942\ncp942c\ncp943\ncp943c\ncp948\ncp949\ncp949c\ncp950\ncp964\ncp970\ncpeuccn\ncpibm284\ncpibm285\ncpibm297\ncpibm37\ncs-ebcdic-cp-ca\ncs-ebcdic-cp-nl\ncs-ebcdic-cp-us\ncs-ebcdic-cp-wt\ncsascii\ncsbig5\ncscesu-8\ncseuckr\ncseucpkdfmtjapanese\ncshalfwidthkatakana\ncsibm037\ncsibm278\ncsibm284\ncsibm285\ncsibm290\ncsibm297\ncsibm420\ncsibm424\ncsibm500\ncsibm857\ncsibm860\ncsibm861\ncsibm862\ncsibm863\ncsibm864\ncsibm865\ncsibm866\ncsibm868\ncsibm869\ncsibm870\ncsibm871\ncsiso153gost1976874\ncsiso159jisx02121990\ncsiso2022cn\ncsiso2022jp\ncsiso2022jp2\ncsiso2022kr\ncsiso87jisx0208\ncsiso885915\ncsiso885916\ncsisolatin0\ncsisolatin1\ncsisolatin2\ncsisolatin3\ncsisolatin4\ncsisolatin5\ncsisolatin9\ncsisolatinarabic\ncsisolatincyrillic\ncsisolatingreek\ncsisolatinhebrew\ncsjisencoding\ncskoi8r\ncspc850multilingual\ncspc862latinhebrew\ncspc8codepage437\ncspcp852\ncspcp855\ncsshiftjis\ncswindows31j\ncyrillic\nebcdic-cp-ar1\nebcdic-cp-ar2\nebcdic-cp-bh\nebcdic-cp-ca\nebcdic-cp-ch\nebcdic-cp-fr\nebcdic-cp-gb\nebcdic-cp-he\nebcdic-cp-is\nebcdic-cp-nl\nebcdic-cp-roece\nebcdic-cp-se\nebcdic-cp-us\nebcdic-cp-wt\nebcdic-cp-yu\nebcdic-de-273+euro\nebcdic-dk-277+euro\nebcdic-es-284+euro\nebcdic-fi-278+euro\nebcdic-fr-277+euro\nebcdic-gb\nebcdic-gb-285+euro\nebcdic-international-500+euro\nebcdic-it-280+euro\nebcdic-jp-kana\nebcdic-no-277+euro\nebcdic-s-871+euro\nebcdic-se-278+euro\nebcdic-sv\nebcdic-us-037+euro\necma-114\necma-118\nelot_928\neuc-cn\neuc-jp\neuc-jp-linux\neuc-kr\neuc-tw\neuc_cn\neuc_jp\neuc_jp_linux\neuc_jp_solaris\neuc_kr\neuc_tw\neuccn\neucjis\neucjp\neucjp-open\neuckr\neuctw\nextended_unix_code_packed_format_for_japanese\ngb18030\ngb18030-2022\ngb2312\ngb2312-1980\ngb2312-80\ngbk\ngreek\ngreek8\nhebrew\nibm-037\nibm-1006\nibm-1025\nibm-1026\nibm-1046\nibm-1047\nibm-1089\nibm-1097\nibm-1098\nibm-1112\nibm-1122\nibm-1123\nibm-1124\nibm-1129\nibm-1140\nibm-1141\nibm-1142\nibm-1143\nibm-1144\nibm-1145\nibm-1146\nibm-1147\nibm-1148\nibm-1149\nibm-1166\nibm-1252\nibm-1364\nibm-1381\nibm-1383\nibm-273\nibm-277\nibm-278\nibm-280\nibm-284\nibm-285\nibm-290\nibm-29626c\nibm-297\nibm-300\nibm-33722\nibm-33722_vascii_vpua\nibm-37\nibm-420\nibm-424\nibm-437\nibm-500\nibm-5050\nibm-737\nibm-775\nibm-813\nibm-819\nibm-833\nibm-834\nibm-838\nibm-850\nibm-852\nibm-855\nibm-856\nibm-857\nibm-858\nibm-860\nibm-861\nibm-862\nibm-863\nibm-864\nibm-865\nibm-866\nibm-868\nibm-869\nibm-870\nibm-871\nibm-874\nibm-875\nibm-912\nibm-913\nibm-914\nibm-915\nibm-916\nibm-918\nibm-920\nibm-921\nibm-922\nibm-923\nibm-930\nibm-932\nibm-933\nibm-935\nibm-937\nibm-939\nibm-942\nibm-942c\nibm-943\nibm-943c\nibm-948\nibm-949\nibm-949c\nibm-950\nibm-964\nibm-970\nibm-euccn\nibm-eucjp\nibm-euckr\nibm-euctw\nibm-thai\nibm00858\nibm01140\nibm01141\nibm01142\nibm01143\nibm01144\nibm01145\nibm01146\nibm01147\nibm01148\nibm01149\nibm037\nibm1006\nibm1025\nibm1026\nibm1046\nibm1047\nibm1089\nibm1097\nibm1098\nibm1112\nibm1122\nibm1123\nibm1124\nibm1129\nibm1140\nibm1141\nibm1142\nibm1143\nibm1144\nibm1145\nibm1146\nibm1147\nibm1148\nibm1149\nibm1166\nibm1252\nibm1364\nibm1381\nibm1383\nibm273\nibm277\nibm278\nibm280\nibm284\nibm285\nibm290\nibm29626c\nibm297\nibm300\nibm33722\nibm367\nibm420\nibm424\nibm437\nibm500\nibm737\nibm775\nibm813\nibm819\nibm833\nibm834\nibm838\nibm850\nibm852\nibm855\nibm856\nibm857\nibm858\nibm860\nibm861\nibm862\nibm863\nibm864\nibm865\nibm866\nibm868\nibm869\nibm870\nibm871\nibm874\nibm875\nibm912\nibm913\nibm914\nibm915\nibm916\nibm918\nibm920\nibm921\nibm922\nibm923\nibm930\nibm932\nibm933\nibm935\nibm937\nibm939\nibm942\nibm942c\nibm943\nibm943c\nibm948\nibm949\nibm949c\nibm950\nibm964\nibm970\nibmeuccn\niscii\niscii91\niso-10646-ucs-2\niso-2022-cn\niso-2022-cn-cns\niso-2022-cn-gb\niso-2022-jp\niso-2022-jp-2\niso-2022-kr\niso-8859-1\niso-8859-11\niso-8859-13\niso-8859-15\niso-8859-16\niso-8859-2\niso-8859-3\niso-8859-4\niso-8859-5\niso-8859-6\niso-8859-7\niso-8859-8\niso-8859-9\niso-ir-100\niso-ir-101\niso-ir-109\niso-ir-110\niso-ir-126\niso-ir-127\niso-ir-138\niso-ir-144\niso-ir-148\niso-ir-153\niso-ir-159\niso-ir-226\niso-ir-6\niso-ir-87\niso2022cn\niso2022cn_cns\niso2022cn_gb\niso2022jp\niso2022jp2\niso2022kr\niso646-us\niso8859-1\niso8859-13\niso8859-15\niso8859-2\niso8859-3\niso8859-4\niso8859-5\niso8859-6\niso8859-7\niso8859-8\niso8859-9\niso8859_1\niso8859_11\niso8859_13\niso8859_15\niso8859_15_fdis\niso8859_16\niso8859_2\niso8859_3\niso8859_4\niso8859_5\niso8859_6\niso8859_7\niso8859_8\niso8859_9\niso_646.irv:1983\niso_646.irv:1991\niso_8859-1\niso_8859-13\niso_8859-15\niso_8859-16\niso_8859-16:2001\niso_8859-1:1987\niso_8859-2\niso_8859-2:1987\niso_8859-3\niso_8859-3:1988\niso_8859-4\niso_8859-4:1988\niso_8859-5\niso_8859-5:1988\niso_8859-6\niso_8859-6:1987\niso_8859-7\niso_8859-7:1987\niso_8859-8\niso_8859-8:1988\niso_8859-9\niso_8859-9:1989\niso_8859_1\njis\njis0201\njis0208\njis0212\njis_c6226-1983\njis_encoding\njis_x0201\njis_x0208-1983\njis_x0212-1990\njisautodetect\njohab\nkoi8\nkoi8-r\nkoi8-u\nkoi8_r\nkoi8_u\nks_c_5601-1987\nksc5601\nksc5601-1987\nksc5601-1992\nksc5601_1987\nksc5601_1992\nksc_5601\nl1\nl10\nl2\nl3\nl4\nl5\nl9\nlatin-9\nlatin0\nlatin1\nlatin10\nlatin2\nlatin3\nlatin4\nlatin5\nlatin9\nmacarabic\nmaccentraleurope\nmaccroatian\nmaccyrillic\nmacdingbat\nmacgreek\nmachebrew\nmaciceland\nmacroman\nmacromania\nmacsymbol\nmacthai\nmacturkish\nmacukraine\nms-874\nms1361\nms50220\nms50221\nms874\nms932\nms932-0213\nms932:2004\nms932_0213\nms936\nms949\nms950\nms950_hkscs\nms950_hkscs_xp\nms_936\nms_949\nms_kanji\npc-multilingual-850+euro\npck\nshift-jis\nshift_jis\nshift_jis:2004\nshift_jis_0213:2004\nsjis\nsjis-0213\nsjis:2004\nsjis_0213\nsjis_0213:2004\nst_sev_358-88\nsun_eu_greek\ntis-620\ntis620\ntis620.2533\nunicode\nunicode-1-1-utf-8\nunicodebig\nunicodebigunmarked\nunicodelittle\nunicodelittleunmarked\nus\nus-ascii\nutf-16\nutf-16be\nutf-16le\nutf-32\nutf-32be\nutf-32be-bom\nutf-32le\nutf-32le-bom\nutf-8\nutf16\nutf32\nutf8\nutf_16\nutf_16be\nutf_16le\nutf_32\nutf_32be\nutf_32be_bom\nutf_32le\nutf_32le_bom\nwindows-1250\nwindows-1251\nwindows-1252\nwindows-1253\nwindows-1254\nwindows-1255\nwindows-1256\nwindows-1257\nwindows-1258\nwindows-31j\nwindows-437\nwindows-874\nwindows-932\nwindows-932-0213\nwindows-932:2004\nwindows-936\nwindows-949\nwindows-950\nwindows-iso2022jp\nwindows949\nx-big5-hkscs-2001\nx-big5-solaris\nx-euc-cn\nx-euc-jp\nx-euc-jp-linux\nx-euc-tw\nx-eucjp\nx-eucjp-open\nx-ibm1006\nx-ibm1025\nx-ibm1046\nx-ibm1097\nx-ibm1098\nx-ibm1112\nx-ibm1122\nx-ibm1123\nx-ibm1124\nx-ibm1129\nx-ibm1166\nx-ibm1364\nx-ibm1381\nx-ibm1383\nx-ibm29626c\nx-ibm300\nx-ibm33722\nx-ibm737\nx-ibm833\nx-ibm834\nx-ibm856\nx-ibm874\nx-ibm875\nx-ibm921\nx-ibm922\nx-ibm930\nx-ibm932\nx-ibm933\nx-ibm935\nx-ibm937\nx-ibm939\nx-ibm942\nx-ibm942c\nx-ibm943\nx-ibm943c\nx-ibm948\nx-ibm949\nx-ibm949c\nx-ibm950\nx-ibm964\nx-ibm970\nx-iscii91\nx-iso-2022-cn-cns\nx-iso-2022-cn-gb\nx-iso-8859-11\nx-jis0208\nx-jisautodetect\nx-johab\nx-macarabic\nx-maccentraleurope\nx-maccroatian\nx-maccyrillic\nx-macdingbat\nx-macgreek\nx-machebrew\nx-maciceland\nx-macroman\nx-macromania\nx-macsymbol\nx-macthai\nx-macturkish\nx-macukraine\nx-ms932_0213\nx-ms950-hkscs\nx-ms950-hkscs-xp\nx-mswin-936\nx-pck\nx-sjis\nx-sjis_0213\nx-utf-16be\nx-utf-16le\nx-utf-16le-bom\nx-utf-32be\nx-utf-32be-bom\nx-utf-32le\nx-utf-32le-bom\nx-windows-50220\nx-windows-50221\nx-windows-874\nx-windows-949\nx-windows-950\nx-windows-iso2022jp\nx0201\nx0208\nx0212"
	if got != expected {
		t.Fatalf("target JDK21 bundled charset registry changed: got %q expected %q", got, expected)
	}
	t.Logf("JDK21 bundled charset name/alias count=%d", len(strings.Split(got, "\n")))
}
