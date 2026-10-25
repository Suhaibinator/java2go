package stdjava

import (
	"sync"
	"unicode/utf16"
)

type systemPropertyState struct {
	sync.Mutex
	installed   bool
	profile     SystemPropertyProfile
	values      map[string]*JavaString
	unavailable map[string]string
}

var systemProperties systemPropertyState

func systemPropertyText(value string) *JavaString {
	return NewJavaStringUTF16(utf16.Encode([]rune(value)))
}
func systemPropertyNativeKey(value string) string {
	return JavaStringSwitchKey(systemPropertyText(value))
}

func systemPropertyKey(key *JavaString) string {
	if key == nil {
		panic(NewJavaNullPointerExceptionMessage(JavaStringLiteralUTF16(utf16.Encode([]rune("key can't be null")))))
	}
	if len(key.units) == 0 {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16(utf16.Encode([]rune("key can't be empty")))))
	}
	return JavaStringSwitchKey(key)
}

// Called with the state locked. The profile is installed once; later property
// mutation cannot alter its identity or silently select another reference JDK.
func (state *systemPropertyState) ensureProfile(profile SystemPropertyProfile) {
	if state.installed {
		if state.profile != profile {
			systemPropertyProfileFailure("conflicting reference metadata in one process")
		}
		return
	}
	profile.validate()
	values := map[string]*JavaString{}
	unavailable := map[string]string{}
	for name, value := range profile.entries() {
		key := systemPropertyNativeKey(name)
		if value == "" {
			unavailable[key] = systemPropertyUnavailable(name)
		} else {
			values[key] = systemPropertyText(value)
		}
	}
	for name, value := range map[string]string{"java.specification.version": "21", "java.class.version": systemPropertyTargetClassVersion, "file.encoding": "UTF-8"} {
		values[systemPropertyNativeKey(name)] = systemPropertyText(value)
	}
	// This finite profile has no proven initial facts for these standard
	// runtime defaults. Do not report them as absent application properties or
	// admit configuration whose dependent platform service is not modeled.
	for _, name := range []string{"java.class.path", "java.library.path", "java.io.tmpdir", "native.encoding", "stdout.encoding", "stderr.encoding"} {
		unavailable[systemPropertyNativeKey(name)] = systemPropertyUnavailable(name)
	}
	for name, value := range systemPropertyStartupHost.values {
		values[systemPropertyNativeKey(name)] = systemPropertyText(value)
	}
	for name, reason := range systemPropertyStartupHost.unavailable {
		unavailable[systemPropertyNativeKey(name)] = reason
	}
	state.profile, state.values, state.unavailable, state.installed = profile, values, unavailable, true
}

func (state *systemPropertyState) requireAvailable(key string) {
	if reason, missing := state.unavailable[key]; missing {
		systemPropertyProfileFailure(reason)
	}
}

func (state *systemPropertyState) get(profile SystemPropertyProfile, key, fallback *JavaString) *JavaString {
	identity := systemPropertyKey(key)
	state.Lock()
	defer state.Unlock()
	state.ensureProfile(profile)
	state.requireAvailable(identity)
	if value, found := state.values[identity]; found {
		return value
	}
	return fallback
}
func (state *systemPropertyState) set(profile SystemPropertyProfile, key, value *JavaString) *JavaString {
	identity := systemPropertyKey(key)
	if value == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	state.Lock()
	defer state.Unlock()
	state.ensureProfile(profile)
	state.requireAvailable(identity)
	previous := state.values[identity]
	state.values[identity] = value
	return previous
}
func (state *systemPropertyState) clear(profile SystemPropertyProfile, key *JavaString) *JavaString {
	identity := systemPropertyKey(key)
	state.Lock()
	defer state.Unlock()
	state.ensureProfile(profile)
	state.requireAvailable(identity)
	previous := state.values[identity]
	delete(state.values, identity)
	return previous
}

func SystemGetProperty(profile SystemPropertyProfile, key *JavaString) *JavaString {
	return systemProperties.get(profile, key, nil)
}
func SystemGetPropertyDefault(profile SystemPropertyProfile, key, fallback *JavaString) *JavaString {
	return systemProperties.get(profile, key, fallback)
}
func SystemSetProperty(profile SystemPropertyProfile, key, value *JavaString) *JavaString {
	return systemProperties.set(profile, key, value)
}
func SystemClearProperty(profile SystemPropertyProfile, key *JavaString) *JavaString {
	return systemProperties.clear(profile, key)
}

// Query only already-published property state for another platform service.
// A URL-only program does not acquire an artificial JDK profile or property set.
func systemPropertyConfigured(name string) *JavaString {
	systemProperties.Lock()
	defer systemProperties.Unlock()
	return systemProperties.values[systemPropertyNativeKey(name)]
}
