package stdjava

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

// SystemPropertyProfile contains facts supplied at build/transpile setup from
// an explicitly selected reference JDK. It is not discovered by the Go program.
// Empty optional identity fields mean unavailable metadata, not a made-up JVM.
// ReleaseSHA256 binds generated call sites to the same immutable input cohort.
// JAVA_HOME/release supplies JAVA_VERSION, JAVA_VERSION_DATE, JAVA_RUNTIME_VERSION and IMPLEMENTOR
// for JavaVersion, VersionDate, RuntimeVersion and Vendor. Optional identity fields require
// corresponding explicit JAVA_RUNTIME_*, JAVA_VM_* or JAVA_SPECIFICATION_* facts;
// JavaHome requires JAVA_RUNTIME_HOME, not a guessed wrapper/distribution path.
// Unsupported identity reads/writes fail explicitly when that fact is absent.
// The semantic Java21/specification and class65 target facts are separate from
// these runtime identity strings. Host OS/user/path values are captured once
// through host APIs, never imported from reference release OS_NAME/OS_ARCH.
type SystemPropertyProfile struct {
	ReleaseSHA256, JavaVersion, VersionDate, RuntimeVersion, Vendor, JavaHome string
	RuntimeName, VendorURL, VMName, VMVersion, VMVendor                       string
	SpecificationName, SpecificationVendor                                    string
	VMSpecificationVersion, VMSpecificationName, VMSpecificationVendor        string
}

const systemPropertyTargetFeature = 21
const systemPropertyTargetClassVersion = "65.0"

func systemPropertyProfileFailure(message string) {
	panic(NewUnsupportedOperationException("System property runtime profile: " + message))
}

func systemPropertyVersionFeature(version string) (int, bool) {
	for _, character := range version {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return 0, false
		}
	}
	end := 0
	for end < len(version) && version[end] >= '0' && version[end] <= '9' {
		end++
	}
	if end == 0 || (end < len(version) && version[end] != '.' && version[end] != '-' && version[end] != '+') {
		return 0, false
	}
	value, err := strconv.Atoi(version[:end])
	return value, err == nil
}

func (profile SystemPropertyProfile) validate() {
	hash, err := hex.DecodeString(profile.ReleaseSHA256)
	if err != nil || len(hash) != 32 {
		systemPropertyProfileFailure("required selected JAVA_HOME/release hash is unavailable")
	}
	feature, ok := systemPropertyVersionFeature(profile.JavaVersion)
	if !ok || feature != systemPropertyTargetFeature {
		systemPropertyProfileFailure("required reference Java21 JAVA_VERSION is unavailable or incompatible")
	}
	if profile.RuntimeVersion != "" {
		if feature, ok := systemPropertyVersionFeature(profile.RuntimeVersion); !ok || feature != systemPropertyTargetFeature {
			systemPropertyProfileFailure("reference JAVA_RUNTIME_VERSION conflicts with Java21 target")
		}
	}
	if profile.VersionDate != "" {
		if _, err := time.Parse("2006-01-02", profile.VersionDate); err != nil {
			systemPropertyProfileFailure("reference JAVA_VERSION_DATE is invalid")
		}
	}
	for _, field := range []string{profile.JavaVersion, profile.VersionDate, profile.RuntimeVersion, profile.Vendor, profile.JavaHome, profile.RuntimeName, profile.VendorURL, profile.VMName, profile.VMVersion, profile.VMVendor, profile.SpecificationName, profile.SpecificationVendor, profile.VMSpecificationVersion, profile.VMSpecificationName, profile.VMSpecificationVendor} {
		if !utf8.ValidString(field) {
			systemPropertyProfileFailure("identity metadata is not valid UTF8")
		}
	}
}

func (profile SystemPropertyProfile) entries() map[string]string {
	return map[string]string{
		"java.version": profile.JavaVersion, "java.version.date": profile.VersionDate, "java.specification.maintenance.version": "", "java.runtime.version": profile.RuntimeVersion,
		"java.vendor": profile.Vendor, "java.home": profile.JavaHome, "java.runtime.name": profile.RuntimeName,
		"java.vendor.url": profile.VendorURL, "java.vm.name": profile.VMName, "java.vm.version": profile.VMVersion, "java.vm.vendor": profile.VMVendor,
		"java.specification.name": profile.SpecificationName, "java.specification.vendor": profile.SpecificationVendor,
		"java.vm.specification.version": profile.VMSpecificationVersion, "java.vm.specification.name": profile.VMSpecificationName, "java.vm.specification.vendor": profile.VMSpecificationVendor,
	}
}

func systemPropertyUnavailable(name string) string {
	return fmt.Sprintf("provided identity/host metadata required for %s", name)
}
