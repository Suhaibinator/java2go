package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Explicit test reference facts are the pinned release fixture, not runtime
// defaults inferred from a language feature or from the Go toolchain.
func systemPropertyTestProfile() SystemPropertyProfile {
	return SystemPropertyProfile{
		ReleaseSHA256: "f61d466aaef05ecd3e431d2cced84792613ee2ef2e20bd7168f4c8d1960c2273", JavaVersion: "21.0.6", RuntimeVersion: "21.0.6+8-LTS-188", Vendor: "Oracle Corporation", JavaHome: "/reference/jdk21",
	}
}
func systemPropertyTestFailure(call func()) (failure any) {
	defer func() { failure = recover() }()
	call()
	return nil
}

func TestSystemPropertyStateAndReferenceIdentity(t *testing.T) {
	var state systemPropertyState
	profile := systemPropertyTestProfile()
	key := systemPropertyText("contract.key")
	alias := systemPropertyText("contract.key")
	first := systemPropertyText("first")
	second := systemPropertyText("second")
	fallback := systemPropertyText("fallback")
	if state.get(profile, key, nil) != nil || state.get(profile, key, fallback) != fallback {
		t.Fatal("missing/default reference changed")
	}
	if state.set(profile, key, first) != nil || state.get(profile, alias, nil) != first || state.get(profile, key, fallback) != first {
		t.Fatal("set/equal key/get reference changed")
	}
	if state.set(profile, alias, second) != first || state.get(profile, key, nil) != second {
		t.Fatal("prior value identity changed")
	}
	if state.clear(profile, alias) != second || state.clear(profile, key) != nil || state.get(profile, key, nil) != nil {
		t.Fatal("clear/absence changed")
	}
	empty := NewJavaStringUTF16(nil)
	if state.set(profile, key, empty) != nil || state.get(profile, key, fallback) != empty || state.clear(profile, key) != empty {
		t.Fatal("empty value was treated as absent")
	}
	whitespace := systemPropertyText(" ")
	state.set(profile, whitespace, first)
	if state.get(profile, whitespace, nil) != first {
		t.Fatal("nonempty whitespace key rejected")
	}
}

func TestSystemPropertyUTF16KeysAndValues(t *testing.T) {
	var state systemPropertyState
	profile := systemPropertyTestProfile()
	high := NewJavaStringUTF16([]uint16{0xd800})
	low := NewJavaStringUTF16([]uint16{0xdc00})
	replacement := NewJavaStringUTF16([]uint16{0xfffd})
	nul := NewJavaStringUTF16([]uint16{0})
	pair := NewJavaStringUTF16([]uint16{0xd800, 0xdc00})
	keys := []*JavaString{high, low, replacement, nul, pair}
	values := []*JavaString{pair, nul, high, low, replacement}
	for i, key := range keys {
		if state.set(profile, key, values[i]) != nil {
			t.Fatal("distinct UTF16 keys aliased")
		}
	}
	for i, key := range keys {
		alias := NewJavaStringUTF16(key.UTF16Copy())
		if state.get(profile, alias, nil) != values[i] || state.clear(profile, alias) != values[i] {
			t.Fatal("UTF16 key/value units or reference changed")
		}
	}
}

func TestSystemPropertyJDKValidationOrder(t *testing.T) {
	profile := systemPropertyTestProfile()
	key := systemPropertyText("contract.validation")
	empty := NewJavaStringUTF16(nil)
	for _, row := range []struct {
		name, kind, message string
		call                func(*systemPropertyState)
	}{
		{"get-null", "NullPointerException", "key can't be null", func(s *systemPropertyState) { s.get(profile, nil, nil) }},
		{"get-empty", "IllegalArgumentException", "key can't be empty", func(s *systemPropertyState) { s.get(profile, empty, nil) }},
		{"set-null-key-and-value", "NullPointerException", "key can't be null", func(s *systemPropertyState) { s.set(profile, nil, nil) }},
		{"set-empty-key-and-null-value", "IllegalArgumentException", "key can't be empty", func(s *systemPropertyState) { s.set(profile, empty, nil) }},
		{"set-null-value", "NullPointerException", "", func(s *systemPropertyState) { s.set(profile, key, nil) }},
		{"clear-null", "NullPointerException", "key can't be null", func(s *systemPropertyState) { s.clear(profile, nil) }},
		{"clear-empty", "IllegalArgumentException", "key can't be empty", func(s *systemPropertyState) { s.clear(profile, empty) }},
	} {
		t.Run(row.name, func(t *testing.T) {
			var state systemPropertyState
			failure := systemPropertyTestFailure(func() { row.call(&state) })
			if !CaughtAs(failure, row.kind) {
				t.Fatalf("JDK exception type=%T want=%s", failure, row.kind)
			}
			message := JavaThrowableMessageDefault(failure)
			if row.message == "" {
				if message != nil {
					t.Fatal("null value NPE message must be Java null")
				}
			} else if message == nil || !message.Equals(systemPropertyText(row.message)) {
				t.Fatalf("JDK message=%v want=%q", message, row.message)
			}
			if state.installed {
				t.Fatal("invalid argument reached profile/store initialization")
			}
		})
	}
}

func TestSystemPropertyDefaultsAreMutableEntries(t *testing.T) {
	var state systemPropertyState
	profile := systemPropertyTestProfile()
	version := systemPropertyText("java.version")
	fallback := systemPropertyText("fallback")
	first := state.get(profile, version, nil)
	if first == nil || !first.Equals(systemPropertyText(profile.JavaVersion)) || state.get(profile, version, nil) != first {
		t.Fatal("declared version/profile identity missing")
	}
	if state.clear(profile, version) != first || state.get(profile, version, nil) != nil || state.get(profile, version, fallback) != fallback {
		t.Fatal("cleared target default reappeared")
	}
	changed := systemPropertyText("user override")
	if state.set(profile, version, changed) != nil || state.get(profile, version, nil) != changed {
		t.Fatal("version override was ignored")
	}
	if !state.get(profile, systemPropertyText("java.specification.version"), nil).Equals(systemPropertyText("21")) || !state.get(profile, systemPropertyText("java.class.version"), nil).Equals(systemPropertyText("65.0")) {
		t.Fatal("semantic target facts changed with runtime-version property")
	}
}

func TestSystemPropertyMissingAndConflictingProfile(t *testing.T) {
	key := systemPropertyText("java.version")
	profile := systemPropertyTestProfile()
	for _, row := range []struct {
		name    string
		profile SystemPropertyProfile
	}{
		{"missing", SystemPropertyProfile{}},
		{"bad-hash", SystemPropertyProfile{ReleaseSHA256: "not a hash", JavaVersion: "21.0.6"}},
		{"wrong-target", SystemPropertyProfile{ReleaseSHA256: profile.ReleaseSHA256, JavaVersion: "17.0.1"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			var state systemPropertyState
			failure := systemPropertyTestFailure(func() { state.get(row.profile, key, nil) })
			if !CaughtAs(failure, "UnsupportedOperationException") || state.installed {
				t.Fatalf("missing/invalid profile invented identity: %T", failure)
			}
		})
	}
	var state systemPropertyState
	value := state.get(profile, key, nil)
	changed := profile
	changed.JavaVersion = "21.0.7"
	if failure := systemPropertyTestFailure(func() { state.get(changed, key, nil) }); !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatalf("mixed profile accepted: %T", failure)
	}
	if state.get(profile, key, nil) != value {
		t.Fatal("profile conflict modified state")
	}
	for _, name := range []string{"java.vm.name", "java.runtime.name", "java.version.date", "java.specification.maintenance.version", "java.class.path", "java.library.path", "java.io.tmpdir", "native.encoding", "stdout.encoding", "stderr.encoding"} {
		if failure := systemPropertyTestFailure(func() { state.get(profile, systemPropertyText(name), nil) }); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("unprovided VM/runtime identity fabricated for %s", name)
		}
	}
	provided := profile
	provided.RuntimeName = "explicitly supplied runtime"
	var other systemPropertyState
	if !other.get(provided, systemPropertyText("java.runtime.name"), nil).Equals(systemPropertyText(provided.RuntimeName)) {
		t.Fatal("provided identity fact ignored")
	}
}

func TestSystemPropertyConcurrentProcessStore(t *testing.T) {
	profile := systemPropertyTestProfile()
	var state systemPropertyState
	key := systemPropertyText("concurrent.key")
	values := []*JavaString{systemPropertyText("first"), NewJavaStringUTF16([]uint16{0xd800}), systemPropertyText("")}
	domain := map[*JavaString]bool{nil: true}
	for _, value := range values {
		domain[value] = true
	}
	var workers sync.WaitGroup
	var errors atomic.Int32
	for worker := 0; worker < 6; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			defer func() {
				if recover() != nil {
					errors.Add(1)
				}
			}()
			for i := 0; i < 128; i++ {
				switch (worker + i) % 3 {
				case 0:
					if !domain[state.set(profile, key, values[worker%len(values)])] {
						errors.Add(1)
					}
				case 1:
					if !domain[state.get(profile, key, nil)] {
						errors.Add(1)
					}
				case 2:
					if !domain[state.clear(profile, key)] {
						errors.Add(1)
					}
				}
			}
		}(worker)
	}
	workers.Wait()
	if errors.Load() != 0 {
		t.Fatalf("concurrent reference/state failures=%d", errors.Load())
	}
}

func TestSystemPropertyHostStartupSnapshot(t *testing.T) {
	var state systemPropertyState
	profile := systemPropertyTestProfile()
	t.Chdir(t.TempDir())
	t.Setenv("HOME", "ignored-after-startup")
	for name, value := range systemPropertyStartupHost.values {
		actual := state.get(profile, systemPropertyText(name), nil)
		if actual == nil || !actual.Equals(systemPropertyText(value)) {
			t.Fatalf("startup host property %s changed after chdir/env write", name)
		}
	}
}

func TestSystemPropertyVersionDateMetadata(t *testing.T) {
	profile := systemPropertyTestProfile()
	profile.VersionDate = "2025-01-21"
	var state systemPropertyState
	key := systemPropertyText("java.version.date")
	original := state.get(profile, key, nil)
	if original == nil || !original.Equals(systemPropertyText(profile.VersionDate)) {
		t.Fatal("release date fact not installed")
	}
	if state.clear(profile, key) != original || state.get(profile, key, nil) != nil {
		t.Fatal("cleared date default reappeared")
	}
	changed := systemPropertyText("application override")
	if state.set(profile, key, changed) != nil || state.get(profile, key, nil) != changed {
		t.Fatal("version date was made an immutable special case")
	}
	bad := profile
	bad.VersionDate = "invented"
	var other systemPropertyState
	if failure := systemPropertyTestFailure(func() { other.get(bad, key, nil) }); !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatalf("invalid date fact accepted: %T", failure)
	}
}
