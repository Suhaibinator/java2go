package transpiler

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// This is a data-only setup input. Generated programs never read JAVA_HOME,
// execute Java, or substitute Go's version for the selected Java runtime facts.
// The selected home locates data only: bin wrappers may delegate elsewhere, so
// java.home requires an explicit JAVA_RUNTIME_HOME identity fact.
func selectedSystemPropertyProfile() (map[string]string, error) {
	home := os.Getenv("JAVA_HOME")
	if home == "" {
		return nil, fmt.Errorf("system property runtime profile requires explicit JAVA_HOME/release")
	}
	home, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("system property runtime profile home: %w", err)
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return nil, fmt.Errorf("system property runtime profile home: %w", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "release"))
	if err != nil {
		return nil, fmt.Errorf("system property runtime profile release: %w", err)
	}
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("system property runtime profile release must be UTF8")
	}
	facts := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("malformed System property runtime profile release entry")
		}
		for _, character := range name {
			if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
				return nil, fmt.Errorf("invalid System property runtime profile release key")
			}
		}
		if _, duplicate := facts[name]; duplicate {
			return nil, fmt.Errorf("duplicate System property runtime profile release key %s", name)
		}
		value, err = strconv.Unquote(value)
		if err != nil {
			return nil, fmt.Errorf("invalid System property runtime profile release value for %s", name)
		}
		facts[name] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("system property runtime profile release: %w", err)
	}
	version := facts["JAVA_VERSION"]
	end := 0
	for end < len(version) && version[end] >= '0' && version[end] <= '9' {
		end++
	}
	if end == 0 || version[:end] != "21" || (end < len(version) && version[end] != '.' && version[end] != '-' && version[end] != '+') {
		return nil, fmt.Errorf("system property runtime profile requires reference JAVA_VERSION 21")
	}
	for _, name := range []string{"JAVA_VERSION", "JAVA_RUNTIME_VERSION"} {
		value := facts[name]
		for _, character := range value {
			if unicode.IsSpace(character) || unicode.IsControl(character) {
				return nil, fmt.Errorf("invalid System property runtime profile version for %s", name)
			}
		}
		if name == "JAVA_RUNTIME_VERSION" && value != "" {
			end := 0
			for end < len(value) && value[end] >= '0' && value[end] <= '9' {
				end++
			}
			if end == 0 || value[:end] != "21" || (end < len(value) && value[end] != '.' && value[end] != '-' && value[end] != '+') {
				return nil, fmt.Errorf("contradictory System property runtime profile JAVA_RUNTIME_VERSION")
			}
		}
	}
	if date := facts["JAVA_VERSION_DATE"]; date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, fmt.Errorf("invalid System property runtime profile JAVA_VERSION_DATE")
		}
	}
	digest := sha256.Sum256(raw)
	return map[string]string{
		"ReleaseSHA256": hex.EncodeToString(digest[:]), "VersionDate": facts["JAVA_VERSION_DATE"], "JavaVersion": version, "RuntimeVersion": facts["JAVA_RUNTIME_VERSION"], "Vendor": facts["IMPLEMENTOR"], "JavaHome": facts["JAVA_RUNTIME_HOME"],
		"RuntimeName": facts["JAVA_RUNTIME_NAME"], "VendorURL": facts["JAVA_VENDOR_URL"], "VMName": facts["JAVA_VM_NAME"], "VMVersion": facts["JAVA_VM_VERSION"], "VMVendor": facts["JAVA_VM_VENDOR"],
		"SpecificationName": facts["JAVA_SPECIFICATION_NAME"], "SpecificationVendor": facts["JAVA_SPECIFICATION_VENDOR"], "VMSpecificationVersion": facts["JAVA_VM_SPECIFICATION_VERSION"], "VMSpecificationName": facts["JAVA_VM_SPECIFICATION_NAME"], "VMSpecificationVendor": facts["JAVA_VM_SPECIFICATION_VENDOR"],
	}, nil
}

func systemPropertyProfileExpr(ctx Ctx) (ast.Expr, error) {
	facts, err := selectedSystemPropertyProfile()
	if err != nil {
		return nil, err
	}
	profile := &ast.CompositeLit{Type: stdjavaQualifiedExpr("SystemPropertyProfile", ctx)}
	for _, name := range []string{"ReleaseSHA256", "JavaVersion", "VersionDate", "RuntimeVersion", "Vendor", "JavaHome", "RuntimeName", "VendorURL", "VMName", "VMVersion", "VMVendor", "SpecificationName", "SpecificationVendor", "VMSpecificationVersion", "VMSpecificationName", "VMSpecificationVendor"} {
		if facts[name] != "" {
			profile.Elts = append(profile.Elts, metadataKey(name, metadataString(facts[name])))
		}
	}
	return profile, nil
}
