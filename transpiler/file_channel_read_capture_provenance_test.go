package transpiler

import (
	"strconv"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// Expected descriptions are retained from the JDK21 CaptureProbe/CaptureEdges
// runtime oracles and their javap synthetic descriptors, rather than JVM slots.
func TestFileChannelReadCaptureDeclarationOrdinals(t *testing.T) {
	cases := []struct {
		name, method string
		descriptions []string
	}{
		{"first-reference", `static A probe(FileChannel first,FileChannel second,long wide){return()->{mark((int)wide);second.read(null,wide);first.read(null,wide);};}`, []string{`"<parameter2>"`, `"<parameter3>"`}},
		{"instance-this", `A probe(FileChannel channel,long wide){return()->{mark((int)this.fieldPosition);channel.read(null,wide);};}`, []string{`"<parameter1>"`}},
		{"instance-no-this", `A probe(FileChannel channel,long wide){return()->channel.read(null,wide);}`, []string{`"<parameter1>"`}},
		{"instance-field", `A probe(long wide){return()->this.field.read(null,wide);}`, []string{`"this.field"`}},
		{"inner-not-outer", `static A probe(FileChannel channel,long offset,int marker){return()->{mark(marker);check(()->channel.read(null,offset));};}`, []string{`"<parameter1>"`}},
		{"sam-after-captures", `static B probe(FileChannel captured,int marker){return(long wide,FileChannel supplied)->{mark(marker);supplied.read(null,wide);captured.read(null,wide);};}`, []string{`"<parameter4>"`, `"<parameter2>"`}},
		{"constant-omitted", `static A probe(FileChannel channel){final long folded=2+3;return()->{mark((int)folded);channel.read(null,folded);};}`, []string{`"<parameter1>"`}},
		{"dead-capture-retained", `static A probe(FileChannel first,FileChannel channel){return()->{if(false)first.read(null,1);channel.read(null,1);};}`, []string{`"<parameter1>"`, `"<parameter2>"`}},
		{"member-selector", `static A probe(H holder,int value,FileChannel channel){return()->{mark(holder.value);channel.read(null,1);};}`, []string{`"<parameter2>"`}},
		{"method-selector", `static A probe(String prefix,int toString,FileChannel channel){return()->{Runnable ignored=prefix::toString;channel.read(null,1);};}`, []string{`"<parameter2>"`}},
		{"monitor-capture", `static A probe(Object lock,FileChannel channel){return()->{synchronized(lock){channel.read(null,1);}};}`, []string{`"<parameter2>"`}},
		{"repeated-wide", `static A probe(long wide,FileChannel channel){return()->{mark((int)wide);mark((int)wide);channel.read(null,wide);};}`, []string{`"<parameter2>"`}},
		{"catch-shadows-constant", `static A probe(FileChannel channel){try{throw new IllegalStateException();}catch(Exception marker){return()->{mark(marker.hashCode());channel.read(null,1);};}}`, []string{`"<parameter2>"`}},
		{"captured-local", `static A probe(){FileChannel channel=null;return()->channel.read(null,1);}`, []string{`"<parameter1>"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, calls := fileChannelCaptureTestCalls(t, tc.method)
			if len(calls) != len(tc.descriptions) {
				t.Fatalf("read sites: %d", len(calls))
			}
			for i, call := range calls {
				local := fileChannelCaptureTestContext(ctx, call)
				got, ok := fileChannelReadNullDescription(call.ChildByFieldName("object"), local, ctx.currentFile.Source)
				if !ok || got != tc.descriptions[i] {
					t.Errorf("site%d description=%q proven=%v want=%q", i, got, ok, tc.descriptions[i])
				}
			}
		})
	}
}
func TestFileChannelReadCaptureBoxedOffsetOrder(t *testing.T) {
	ctx, calls := fileChannelCaptureTestCalls(t, `static A probe(FileChannel channel,Long offset,ByteBuffer dst){return()->{if(dst==null)mark(1);channel.read(dst,offset);};}`)
	call := calls[0]
	local := fileChannelCaptureTestContext(ctx, call)
	for _, tc := range []struct {
		node    *sitter.Node
		ordinal int
	}{{call.ChildByFieldName("object"), 2}, {invocationArgumentNode(call, 1), 3}} {
		got, ok := fileChannelReadNullDescription(tc.node, local, ctx.currentFile.Source)
		want := strconv.Quote("<parameter" + strconv.Itoa(tc.ordinal) + ">")
		if !ok || got != want {
			t.Errorf("description=%q/%v want=%q", got, ok, want)
		}
	}
}
func TestFileChannelReadCaptureUnprovedLocalHeld(t *testing.T) {
	for name, method := range map[string]string{
		"inferred-constant-type": `static A probe(FileChannel channel){final var folded=2+3;return()->{mark(folded);channel.read(null,1);};}`,
		"lambda-local":           `static A probe(FileChannel captured){return()->{FileChannel local=captured;local.read(null,3);};}`,
		"anonymous-environment":  `static A probe(FileChannel channel){return()->{Object obj=new Object(){public String toString(){return "x";}};channel.read(null,1);};}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, calls := fileChannelCaptureTestCalls(t, method)
			local := fileChannelCaptureTestContext(ctx, calls[0])
			if got, ok := fileChannelReadNullDescription(calls[0].ChildByFieldName("object"), local, ctx.currentFile.Source); ok {
				t.Fatalf("unproved bytecode environment described as %q", got)
			}
		})
	}
	ctx, calls := fileChannelCaptureTestCalls(t, `static A probe(FileChannel captured){return()->captured.read(null,3);}`)
	local := fileChannelCaptureTestContext(ctx, calls[0])
	local.currentFile = nil
	if got, ok := fileChannelReadNullDescription(calls[0].ChildByFieldName("object"), local, ctx.currentFile.Source); ok {
		t.Fatalf("detached declaration described as %q", got)
	}
}
func fileChannelCaptureTestCalls(t *testing.T, method string) (Ctx, []*sitter.Node) {
	t.Helper()
	java := `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;
 class Capture{interface A{void run()throws Exception;}interface B{void run(long wide,FileChannel value)throws Exception;}static class H{int value;}
 FileChannel field;long fieldPosition;static final int marker=99;static int mark(int n){return n;}static void check(A a){} ` + method + `}`
	ctx := ordinaryDemandTestContext(t, map[string]string{"Capture.java": java}, "Capture.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	ctx.className = ctx.currentClass.Class.Name
	ctx.executionContextName = "__java2goExecution"
	var calls []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "read" {
			calls = append(calls, n)
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	return ctx, calls
}
func fileChannelCaptureTestContext(ctx Ctx, call *sitter.Node) Ctx {
	var lambdas []*sitter.Node
	for n := call.Parent(); n != nil; n = n.Parent() {
		if n.Type() == "lambda_expression" {
			lambdas = append(lambdas, n)
		}
		if n.Type() == "method_declaration" {
			break
		}
	}
	for i := len(lambdas) - 1; i >= 0; i-- {
		ctx = contextWithLambdaParameters(ctx, lambdas[i].ChildByFieldName("parameters"), nil, "void", ctx.currentFile.Source)
	}
	return ctx
}

func TestFileChannelReadCaptureSourceAndBinderOwnership(t *testing.T) {
	for name, java := range map[string]string{
		"source": `class FileChannel{int read(Object buffer,long position){return 3;}}class Capture{static Runnable make(FileChannel channel){return()->{channel.read(null,3);};}}`,
		"binder": `class Capture{static <T extends java.nio.channels.FileChannel> Runnable make(T channel){return()->{channel.read(null,3);};}}`,
	} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, java)
			if strings.Contains(out, "FileChannelReadAtVirtualExecution(") {
				t.Fatalf("capture description replaced declaration ownership:\n%s", out)
			}
		})
	}
}
