package stdjava

import (
	"os"
	"strings"
	"sync"
)

var localeDefaultState struct {
	sync.Mutex
	value   *Locale
	format  *Locale
	display *Locale
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
	localeDefaultState.format = locale
	localeDefaultState.display = locale
	localeDefaultState.Unlock()
}
func (locale *Locale) Equals(other any) bool {
	ReferenceRequireNonNull(locale)
	value, ok := other.(*Locale)
	return ok && value != nil && locale.tag == value.tag
}

// Locale.Category selects an independent default; setDefault(Locale) updates all three.
type LocaleCategory struct{ metadata EnumMetadata }

var LocaleCategoryDISPLAY = &LocaleCategory{metadata: NewEnumMetadata("DISPLAY", 0, "java.util.Locale$Category", "java.util.Locale$Category")}
var LocaleCategoryFORMAT = &LocaleCategory{metadata: NewEnumMetadata("FORMAT", 1, "java.util.Locale$Category", "java.util.Locale$Category")}

func (*LocaleCategory) JavaDynamicTypeID() TypeID { return "java.util.Locale$Category" }
func (category *LocaleCategory) JavaEnumMetadata() *EnumMetadata {
	ReferenceRequireNonNull(category)
	return &category.metadata
}
func LocaleGetDefaultCategory(category *LocaleCategory) *Locale {
	ReferenceRequireNonNull(category)
	general := LocaleGetDefault()
	localeDefaultState.Lock()
	defer localeDefaultState.Unlock()
	switch category {
	case LocaleCategoryFORMAT:
		if localeDefaultState.format == nil {
			localeDefaultState.format = general
		}
		return localeDefaultState.format
	case LocaleCategoryDISPLAY:
		if localeDefaultState.display == nil {
			localeDefaultState.display = general
		}
		return localeDefaultState.display
	default:
		panic(NewIllegalArgumentException("unknown locale category"))
	}
}
func LocaleSetDefaultCategory(category *LocaleCategory, locale *Locale) {
	ReferenceRequireNonNull(category)
	ReferenceRequireNonNull(locale)
	localeDefaultState.Lock()
	defer localeDefaultState.Unlock()
	switch category {
	case LocaleCategoryFORMAT:
		localeDefaultState.format = locale
	case LocaleCategoryDISPLAY:
		localeDefaultState.display = locale
	default:
		panic(NewIllegalArgumentException("unknown locale category"))
	}
}
func (locale *Locale) GetLanguageJavaString() *JavaString {
	ReferenceRequireNonNull(locale)
	base, _, _ := locale.tag.Raw()
	text := base.String()
	if text == "und" {
		text = ""
	}
	return dateFormatNativeJavaString(text)
}
func (locale *Locale) GetCountryJavaString() *JavaString {
	ReferenceRequireNonNull(locale)
	_, _, region := locale.tag.Raw()
	text := region.String()
	if text == "ZZ" {
		text = ""
	}
	return dateFormatNativeJavaString(text)
}
func init() {
	RegisterJavaType("java.util.Locale$Category", EnumTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: "java.util.Locale$Category", SimpleName: "Category", HasSimpleName: true, Enum: true})
}
