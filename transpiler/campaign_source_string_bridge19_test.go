package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// These tests compile the actual rendered source declarations, rather than a
// handwritten imitation of their execution callbacks. They intentionally avoid
// process argv/stdout lowering and Object default/Throwable/enum conversion.
// Strict multi-package acceptance remains a separate gate.
func campaignSourceStringBridge19(t *testing.T, class, source, expected, extraGo string) {
	t.Helper()
	if got := campaignRuntimeJavaOracle(t, class, source); got != expected {
		t.Fatalf("invalid JVM oracle: got %q, authored observation %q", got, expected)
	}
	t.Logf("valid JVM source bridge oracle: %q", expected)
	generated := renderGoFileFromJava(t, source)
	temporary := t.TempDir()
	module := "module bridgeprobe\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	driver := fmt.Sprintf(`package main
import("testing";"reflect";"unicode/utf16"; j "github.com/NickyBoy89/java2go/stdjava")
func TestGeneratedSourceBridge(t *testing.T) {
 got := Run()
 if got == nil || !reflect.DeepEqual(got.UTF16Copy(), utf16.Encode([]rune(%q))) { t.Fatalf("generated run returned %%v", got) }
 _ = j.UTF_8
 %s
}
`, expected, extraGo)
	for name, data := range map[string]string{"go.mod": module, "generated.go": generated, "bridge_test.go": driver} {
		if err := os.WriteFile(filepath.Join(temporary, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	campaignStringRequireSuccess(t, "generated source bridge race test", campaignStringBoundedRun(t, "generated source bridge race test", temporary, "go", 90*time.Second, "test", "-race", "-count=1", "-timeout=45s", "-mod=mod", "."))
}

func TestCampaignSourceStringBridge19IdentityNullHostJVM(t *testing.T) {
	campaignSourceStringBridge19(t, "BridgeText", `public class BridgeText {
 String text;
 BridgeText(String text){this.text=text;}
 public synchronized String toString(){return text;}
 public static String run(){
  String payload=new String("\uD800");BridgeText value=new BridgeText(payload);
  String result=String.valueOf((Object)value);
  boolean identity=result==payload;
  int unit=result.charAt(0), encoded=result.getBytes(java.nio.charset.StandardCharsets.UTF_8)[0];
  value.text=null;
  return identity+":"+unit+":"+(String.valueOf((Object)value)==null)+":"+encoded;
 }
}`, "true:55296:true:63", `
 payload:=j.NewJavaStringUTF16([]uint16{0xD800})
 value:=newBridgeText(payload)
 execution:=j.NewExecution()
 guard:=j.MonitorEnterExecution(execution,value)
 gotPointer:=j.JavaStringValueOfExecution(execution,value)
 j.MonitorExitExecution(guard)
 if gotPointer!=payload || payload.CharAt(0)!=0xD800 {t.Fatal("Java callback changed identity or units")}
 native,ok:=any(value).(interface{String() string})
 if !ok {t.Fatal("noncolliding source lacks optional native presentation wrapper")}
 if text:=native.String();text!="?" {t.Fatalf("native UTF8 text %q",text)}
 value.text=nil
 if j.JavaStringValueOfExecution(execution,value)!=nil {t.Fatal("Java callback normalized returned null")}
 if text:=native.String();text!="null" {t.Fatalf("native null presentation %q",text)}
 `)
}

func TestCampaignSourceStringBridge19SelectorCollisionJVM(t *testing.T) {
	campaignSourceStringBridge19(t, "BridgeCollision", `public class BridgeCollision {
 String text=new String("actual");int ordinary;
 public String String(){ordinary++;return "ordinary";}
 public String StringJava2goExecution(){ordinary++;return "impostor";}
 public String toString(){return text;}
 public static String run(){
  BridgeCollision value=new BridgeCollision();
  String result=String.valueOf((Object)value);
  return (result==value.text)+":"+value.ordinary+":"+value.String()+":"+value.StringJava2goExecution();
 }
}`, "true:0:ordinary:impostor", `
 value:=NewBridgeCollision()
 if _,host:=any(value).(interface{String() string});host {t.Fatal("native fmt wrapper occupied ordinary source String selector")}
 if j.JavaStringValueOfExecution(j.NewExecution(),value)!=value.text || value.ordinary!=0 {t.Fatal("registry called ordinary source selector")}
 `)
}

func TestCampaignSourceStringBridge19InheritedReentrantJVM(t *testing.T) {
	campaignSourceStringBridge19(t, "BridgeReentry", `class BridgeBase {
 String text;int calls,depth,selected,mismatch;
 BridgeBase(String text){this.text=text;}
 String select(){return text;}
 public synchronized String toString(){
  calls++;
  if(!Thread.holdsLock(this) || BridgeReentry.context.get()!=BridgeReentry.token)mismatch++;
  if(depth==0){depth=1;String nested=String.valueOf((Object)this);depth=0;if(nested!=text)mismatch++;}
  return select();
 }
}
class BridgeChild extends BridgeBase {
 BridgeChild(String text){super(text);}
 String select(){selected++;return text;}
}
public class BridgeReentry {
 static ThreadLocal<String> context=new ThreadLocal<>();
 static String token=new String("logical-caller");
 public static String run(){
  context.set(token);String payload=new String("\uDFFF");BridgeChild child=new BridgeChild(payload);BridgeBase base=child;
  String result;
  synchronized(child){result=String.valueOf((Object)base);}
  context.remove();
  return (result==payload)+":"+child.calls+":"+child.selected+":"+child.depth+":"+child.mismatch;
 }
}`, "true:2:2:0:0", ``)
}
