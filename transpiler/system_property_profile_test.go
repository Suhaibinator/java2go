package transpiler

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSystemPropertySelectedReleaseFacts(t *testing.T) {
	home := systemPropertyReferenceHome(t)
	t.Setenv("JAVA_HOME", home)
	facts, err := selectedSystemPropertyProfile()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "release"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	if facts["ReleaseSHA256"] != hex.EncodeToString(hash[:]) || facts["JavaVersion"] != "21.0.6" || facts["VersionDate"] != "2025-01-21" || facts["RuntimeVersion"] != "21.0.6+8-LTS-188" || facts["Vendor"] != "Oracle Corporation" {
		t.Fatalf("reference release facts changed: %v", facts)
	}
	for _, name := range []string{"VMName", "VMVersion", "VMVendor", "RuntimeName", "JavaHome"} {
		if facts[name] != "" {
			t.Fatalf("unprovided identity %s invented", name)
		}
	}
	if _, found := facts["os.name"]; found {
		t.Fatal("reference OS metadata confused with Go host")
	}
}

func TestSystemPropertyMalformedReleaseRefusals(t *testing.T) {
	for name, raw := range map[string]string{
		"invalid-date":    "JAVA_VERSION=\"21.0.6\"\nJAVA_VERSION_DATE=\"not-a-date\"\n",
		"missing-version": `IMPLEMENTOR="provided"`,
		"wrong-runtime":   "JAVA_VERSION=\"21.0.6\"\nJAVA_RUNTIME_VERSION=\"17.0.1\"\n",
		"control-version": `JAVA_VERSION="21.0.6\n"`,
		"wrong-target":    `JAVA_VERSION="17.0.1"`,
		"wrong-prefix":    `JAVA_VERSION="210.0.1"`,
		"unquoted":        `JAVA_VERSION=21.0.6`,
		"duplicate":       "JAVA_VERSION=\"21.0.6\"\nJAVA_VERSION=\"21.0.7\"\n",
		"invalid-key":     "JAVA_VERSION=\"21.0.6\"\nnot a key=\"bad\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("JAVA_HOME", home)
			if err := os.WriteFile(filepath.Join(home, "release"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if facts, err := selectedSystemPropertyProfile(); err == nil {
				t.Fatalf("malformed profile admitted: %v", facts)
			}
		})
	}
}

func TestSystemPropertyProfileCodegenUsesSelectedFacts(t *testing.T) {
	home := systemPropertyReferenceHome(t)
	path := filepath.Join(home, "release")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.ReplaceAll(strings.ReplaceAll(string(raw), `JAVA_VERSION="21.0.6"`, `JAVA_VERSION="21.0.77"`), `IMPLEMENTOR="Oracle Corporation"`, `IMPLEMENTOR="explicit test vendor"`))
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JAVA_HOME", home)
	expression, err := systemPropertyProfileExpr(Ctx{})
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := expression.(*ast.CompositeLit)
	if !ok {
		t.Fatal("typed profile literal absent")
	}
	facts := map[string]string{}
	for _, element := range profile.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			t.Fatal("named metadata field absent")
		}
		name, ok := field.Key.(*ast.Ident)
		if !ok {
			t.Fatal("profile field identity absent")
		}
		literal, ok := field.Value.(*ast.BasicLit)
		if !ok {
			t.Fatal("profile field is not an immutable literal")
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		facts[name.Name] = value
	}
	digest := sha256.Sum256(raw)
	if facts["JavaVersion"] != "21.0.77" || facts["Vendor"] != "explicit test vendor" || facts["ReleaseSHA256"] != hex.EncodeToString(digest[:]) {
		t.Fatalf("codegen substituted invented/reference witness facts: %v", facts)
	}
}
