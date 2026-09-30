package stdjava

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Locale carries the language-sensitive casing tag. These operations use the
// Unicode full case mapper, including expansions and context-sensitive rules.
type Locale struct{ tag language.Tag }

var (
	LocaleROOT    = &Locale{language.Und}
	LocaleENGLISH = &Locale{language.English}
	LocaleUS      = &Locale{language.AmericanEnglish}
)

func LocaleForLanguageTag(tag string) *Locale {
	StringRequireNonNull(tag)
	return &Locale{language.Make(tag)}
}
func (*Locale) JavaDynamicTypeID() TypeID { return "Locale" }
func init()                               { RegisterJavaType("Locale", ObjectTypeID) }
func StringToUpperCaseLocale(value string, locale *Locale) string {
	StringRequireNonNull(value)
	ReferenceRequireNonNull(locale)
	// Casers may be stateful; each invocation owns its instance.
	return cases.Upper(locale.tag).String(value)
}
func StringToLowerCaseLocale(value string, locale *Locale) string {
	StringRequireNonNull(value)
	ReferenceRequireNonNull(locale)
	return cases.Lower(locale.tag).String(value)
}
