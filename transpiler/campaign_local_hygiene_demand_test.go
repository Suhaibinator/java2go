package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestCampaignLocalHygieneResourceCleanupJVMParity(t *testing.T) {
	const source = `public class ResourceNames {
 static String log="";
 static final class Resource implements AutoCloseable {
  public void close(){log += "close;";}
 }
 static String work(){
  try(Resource string=new Resource()){
   String value="body;";
   log+=value;
   return value;
  }
 }
 public static String run(){String result=work();return result+log;}
 }`
	want := campaignRuntimeJavaOracle(t, "ResourceNames", source)
	out := renderGoFileFromJava(t, source)
	// Canonical Java String uses *stdjava.JavaString, so this body no longer
	// needs Go's predeclared string type beside the resource binding.
	if !strings.Contains(out, "string := ") {
		t.Fatal("canonical resource binding was not retained")
	}
	runGoTestInTempModule(t, out, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
)
func TestResourceNames(t *testing.T) {
 const want = %q
 got := Run()
 if got == nil { t.Fatal("Run() returned null") }
 units := got.UTF16Copy()
 if !slices.Equal(units, utf16.Encode([]rune(want))) {
  t.Fatalf("JVM %%q != Go %%q", want, string(utf16.Decode(units)))
 }
}`, want))
}

func TestCampaignLocalHygieneImplicitTypeDemandJVMParity(t *testing.T) {
	const source = `class Holder<T>{T item;Holder(T item){this.item=item;}T read(){return item;}}
 public class ImplicitNames {
  static String method(Holder<String> box,int string){return box.read()+string;}
  static String field(Holder<String> box,int string){return box.item+string;}
  static int length(int[] array,int int32,int len){return array.length+int32+len;}
  static long widened(int value,int int64){return value+int64;}
  public static String run(){Holder<String> box=new Holder<String>("x");return method(box,2)+":"+field(box,3)+":"+length(new int[]{1,2},4,5)+":"+widened(6,7);}
 }`
	want := campaignRuntimeJavaOracle(t, "ImplicitNames", source)
	out := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, out, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
)
func TestImplicitNames(t *testing.T) {
 const want = %q
 got := Run()
 if got == nil { t.Fatal("Run() returned null") }
 units := got.UTF16Copy()
 if !slices.Equal(units, utf16.Encode([]rune(want))) {
  t.Fatalf("JVM %%q != Go %%q", want, string(utf16.Decode(units)))
 }
}`, want))
}

func TestLocalHygieneBodyContextIsolation(t *testing.T) {
	helper := setupParseHelper(t, `class Names{void first(){String text="";}void second(int string){}}`)
	owner := overrideBridgeTestScope(t, helper, "Names")
	first := overrideBridgeTestMethod(t, owner, "first")
	second := overrideBridgeTestMethod(t, owner, "second")
	ctx := helper.Ctx.Clone()
	ctx.localBindingBody = first.DeclarationNode.ChildByFieldName("body")
	ctx.localScope = second
	if localIdentifierRequiredByBody("string", ctx) {
		t.Fatal("stale lambda body replaced the current method body")
	}
	switched := classScopeCtx(owner, ctx)
	if switched.localBindingBody != nil {
		t.Fatal("owner switch retained another method's binding body")
	}
}
