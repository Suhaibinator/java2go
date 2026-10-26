package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestFileChannelWriteResourceActualTreeIdentity(t *testing.T) {
	const java = `package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{void run()throws Exception{try(FileChannel resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}}}`
	ctx, source := resourceWriteTestContext(t, "original", "probe.Case", "run", java)
	occurrence := resourceWriteTestCalls(ctx.localScope.DeclarationNode, source)[0].ChildByFieldName("object")
	binding, known := fileChannelReadCaptureDeclaration(occurrence, source)
	if !known || binding.declaration == nil {
		t.Fatal("actual resource declaration missing")
	}
	recordLocalVariableDefinition(ctx, "resource", "FileChannel", "FileChannel")
	if proof, known := fileChannelWriteSourceResourceProof(occurrence, ctx, source); !known || proof.resource == nil || !proof.resource.Equal(binding.declaration) {
		t.Fatal("valid same-tree occurrence lost its declaration certificate")
	}
	for _, row := range []struct {
		name, file     string
		detachedMethod bool
	}{
		{"detached-occurrence", "resource-receiver-test/original/probe/Case.java", false},
		{"detached-method", "resource-receiver-test/original/probe/Case.java", true},
		{"cross-file-twin", "separate-owner/Case.java", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			twinSource := append([]byte(nil), source...)
			if row.name == "cross-file-twin" {
				twinSource = []byte(strings.Replace(string(source), "package probe;", "package other;", 1))
			}
			twin := parsing.SourceFile{Name: row.file, Source: twinSource}
			if err := twin.ParseAST(); err != nil {
				t.Fatal(err)
			}
			twinFile := twin.ParseSymbols()
			if twinFile == nil || twinFile.BaseClass == nil {
				t.Fatal("detached declaration tree unavailable")
			}
			twinMethod := twinFile.BaseClass.FindMethodByName("run", nil)
			if twinMethod == nil || twinMethod.DeclarationNode == nil {
				t.Fatal("detached method unavailable")
			}
			twinOccurrence := resourceWriteTestCalls(twinMethod.DeclarationNode, twin.Source)[0].ChildByFieldName("object")
			if !sameSourceNode(occurrence, twinOccurrence) || occurrence.Equal(twinOccurrence) || !sameSourceNode(ctx.localScope.DeclarationNode, twinMethod.DeclarationNode) || ctx.localScope.DeclarationNode.Equal(twinMethod.DeclarationNode) {
				t.Fatal("control no longer has matching spans and distinct actual trees")
			}
			candidate := ctx.Clone()
			node := twinOccurrence
			argumentSource := twin.Source
			if row.name == "cross-file-twin" && (twinFile.Package == ctx.currentFile.Package || len(twin.Source) != len(source)) {
				t.Fatal("cross-file declaration ownership control lost its distinct package or matching offsets")
			}
			if row.detachedMethod {
				method := *ctx.localScope
				method.DeclarationNode = twinMethod.DeclarationNode
				candidate.localScope = &method
				node = occurrence
				argumentSource = source
			}
			t.Logf("file=%s methodEqual=%t occurrenceEqual=%t matchingSpans=true", row.file, ctx.localScope.DeclarationNode.Equal(twinMethod.DeclarationNode), occurrence.Equal(twinOccurrence))
			if proof, known := fileChannelWriteSourceResourceProof(node, candidate, argumentSource); known || proof.kind != fileChannelWriteReceiverUnavailable {
				t.Fatal("detached tree acquired an actual resource declaration certificate")
			}
		})
	}
	// Parent() may return a distinct Go wrapper for one actual node. Tree-sitter
	// equality, rather than wrapper pointer identity, must retain that valid case.
	var sameMethod *sitter.Node
	for n := occurrence.Parent(); n != nil; n = n.Parent() {
		if n.Equal(ctx.localScope.DeclarationNode) {
			sameMethod = n
			break
		}
	}
	if sameMethod == nil {
		t.Fatal("actual method ancestor missing")
	}
	same := ctx.Clone()
	method := *ctx.localScope
	method.DeclarationNode = sameMethod
	same.localScope = &method
	if _, known := fileChannelWriteSourceResourceProof(occurrence, same, source); !known {
		t.Fatal("same-tree method wrapper refused")
	}
	if _, known := fileChannelWriteSourceResourceProof(nil, ctx, source); known {
		t.Fatal("nil occurrence admitted")
	}
}
