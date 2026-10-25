package stdjava

import "testing"

func systemPropertyURLScope(t *testing.T, handler, delay *JavaString) {
	t.Helper()
	url180ProviderRegistryScope(t)
	profile := systemPropertyTestProfile()
	keys := []*JavaString{systemPropertyText("java.protocol.handler.pkgs"), systemPropertyText("jdk.net.url.delayParsing")}
	values := []*JavaString{handler, delay}
	previous := make([]*JavaString, 2)
	for i, key := range keys {
		previous[i] = SystemGetProperty(profile, key)
		if values[i] == nil {
			SystemClearProperty(profile, key)
		} else {
			SystemSetProperty(profile, key, values[i])
		}
	}
	t.Cleanup(func() {
		for i, key := range keys {
			if previous[i] == nil {
				SystemClearProperty(profile, key)
			} else {
				SystemSetProperty(profile, key, previous[i])
			}
		}
	})
}

func TestSystemPropertyURLConfigurationBoundary(t *testing.T) {
	for _, row := range []struct {
		name           string
		handler, delay *JavaString
		refuse         bool
	}{
		{"default", nil, nil, false},
		{"empty-handler", systemPropertyText(""), nil, false},
		{"default-delay-true", nil, systemPropertyText("true"), false},
		{"default-delay-uppercase", nil, systemPropertyText("TRUE"), false},
		{"default-delay-empty", nil, systemPropertyText(""), false},
		{"unprovided-handler", systemPropertyText("mine"), nil, true},
		{"unprovided-eager-mode", nil, systemPropertyText("false"), true},
		{"unprovided-nonboolean-mode", nil, systemPropertyText("other"), true},
	} {
		t.Run(row.name, func(t *testing.T) {
			systemPropertyURLScope(t, row.handler, row.delay)
			var value *JavaURL
			failure := systemPropertyTestFailure(func() {
				value = NewJavaURLStringExecution(NewExecution(), systemPropertyText("http://Example.invalid/a"))
			})
			if row.refuse {
				if !CaughtAs(failure, "UnsupportedOperationException") {
					t.Fatalf("unprovided URL property configuration silently used: %T value=%v", failure, value)
				}
			} else if failure != nil || value == nil || !value.ToExternalFormJava2goExecution(NewExecution()).Equals(systemPropertyText("http://Example.invalid/a")) {
				t.Fatalf("admitted default URL changed: %T %v", failure, value)
			}
		})
	}
}

func TestSystemPropertyURLNullInputOrdering(t *testing.T) {
	systemPropertyURLScope(t, systemPropertyText("mine"), systemPropertyText("false"))
	failure := systemPropertyTestFailure(func() { NewJavaURLStringExecution(NewExecution(), nil) })
	if !CaughtAs(failure, "MalformedURLException") {
		t.Fatalf("URL null spec must retain original constructor error before configuration: %T", failure)
	}
}
