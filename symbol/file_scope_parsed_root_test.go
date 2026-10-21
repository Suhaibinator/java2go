package symbol_test

import (
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestFileScopeParsedRootIsIndependentAndNilSafe(t *testing.T) {
	var absent *symbol.FileScope
	if absent.ParsedRoot() != nil || (&symbol.FileScope{}).ParsedRoot() != nil {
		t.Fatal("synthetic FileScope acquired parse ownership")
	}
	java := `abstract class R extends java.io.Reader{}`
	first := parsing.SourceFile{Name: "Original.java", Source: []byte(java)}
	if err := first.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file := first.ParseSymbols()
	if file.ParsedRoot() == nil || !file.ParsedRoot().Equal(first.Ast) {
		t.Fatal("ParseSymbols did not retain exact supplied parse root")
	}
	second := parsing.SourceFile{Name: "ByteIdenticalOtherFile.java", Source: []byte(java)}
	if err := second.ParseAST(); err != nil {
		t.Fatal(err)
	}
	twin := second.ParseSymbols()
	if file.ParsedRoot().Equal(twin.ParsedRoot()) {
		t.Fatal("byte-identical separate parse trees collapsed")
	}
	original := file.ParsedRoot()
	file.BaseClass.Class = twin.BaseClass.Class
	file.TopLevelClasses = []*symbol.ClassScope{twin.BaseClass}
	if !file.ParsedRoot().Equal(original) || file.ParsedRoot().Equal(twin.ParsedRoot()) {
		t.Fatal("mutable symbol aliases replaced independent parsed-root certificate")
	}
	copy := *file
	if !copy.ParsedRoot().Equal(original) {
		t.Fatal("ordinary FileScope copy lost original parse identity")
	}
}

func TestFileScopeParsedRootRecoveryAndLifetime(t *testing.T) {
	for _, source := range []string{
		"",
		`abstract class \u0052 extends java.io.Reader{}`,
		`class R{Object x;void update(){(x)=null;}}`,
		`abstract class Empty extends java.io.Reader{}class Derived extends Empty{}`,
	} {
		t.Run(source, func(t *testing.T) {
			file := func() *symbol.FileScope {
				parsed := parsing.SourceFile{Name: "Owned.java", Source: []byte(source)}
				if err := parsed.ParseAST(); err != nil {
					t.Fatal(err)
				}
				symbols := parsed.ParseSymbols()
				if symbols.ParsedRoot() == nil || !symbols.ParsedRoot().Equal(parsed.Ast) {
					t.Fatal("recovery/canonicalization changed actual parsed-root ownership")
				}
				return symbols
			}()
			runtime.GC()
			if file.ParsedRoot() == nil || file.ParsedRoot().Type() != "program" {
				t.Fatal("owned parsed root did not preserve normal tree lifetime")
			}
		})
	}
}
