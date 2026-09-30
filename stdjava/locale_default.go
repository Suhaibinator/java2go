package stdjava

import (
	"os"
	"strings"
	"sync"
)

var localeDefaultState struct {
	sync.Mutex
	value *Locale
}

func LocaleGetDefault() *Locale {
	localeDefaultState.Lock()
	defer localeDefaultState.Unlock()
	if localeDefaultState.value == nil {
		name := os.Getenv("LC_ALL")
		if name == "" {
			name = os.Getenv("LC_MESSAGES")
		}
		if name == "" {
			name = os.Getenv("LANG")
		}
		name = strings.SplitN(name, ".", 2)[0]
		if name == "" || name == "C" || name == "POSIX" {
			localeDefaultState.value = LocaleUS
		} else {
			localeDefaultState.value = LocaleForLanguageTag(strings.ReplaceAll(name, "_", "-"))
		}
	}
	return localeDefaultState.value
}
func LocaleSetDefault(locale *Locale) {
	ReferenceRequireNonNull(locale)
	localeDefaultState.Lock()
	localeDefaultState.value = locale
	localeDefaultState.Unlock()
}
func (locale *Locale) Equals(other any) bool {
	ReferenceRequireNonNull(locale)
	value, ok := other.(*Locale)
	return ok && value != nil && locale.tag == value.tag
}
