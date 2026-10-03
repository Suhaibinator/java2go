package stdjava

import (
	"sync"
	"testing"
)

func preserveLocaleCategories(t *testing.T) {
	t.Helper()
	general := LocaleGetDefault()
	format := LocaleGetDefaultCategory(LocaleCategoryFORMAT)
	display := LocaleGetDefaultCategory(LocaleCategoryDISPLAY)
	t.Cleanup(func() {
		LocaleSetDefault(general)
		LocaleSetDefaultCategory(LocaleCategoryFORMAT, format)
		LocaleSetDefaultCategory(LocaleCategoryDISPLAY, display)
	})
}

func TestLocaleCategoryIndependentDefaultsAndFormatSelection(t *testing.T) {
	preserveLocaleCategories(t)
	LocaleSetDefault(LocaleUS)
	LocaleSetDefaultCategory(LocaleCategoryFORMAT, LocaleENGLISH)
	LocaleSetDefaultCategory(LocaleCategoryDISPLAY, LocaleROOT)
	if LocaleGetDefault() != LocaleUS || LocaleGetDefaultCategory(LocaleCategoryFORMAT) != LocaleENGLISH || LocaleGetDefaultCategory(LocaleCategoryDISPLAY) != LocaleROOT {
		t.Fatal("category mutation changed another default")
	}
	if dateFormatNativeText(LocaleGetDefault().GetLanguageJavaString()) != "en" || dateFormatNativeText(LocaleGetDefault().GetCountryJavaString()) != "US" || dateFormatNativeText(LocaleENGLISH.GetCountryJavaString()) != "" || dateFormatNativeText(LocaleROOT.GetLanguageJavaString()) != "" {
		t.Fatal("language/country accessors inferred missing components")
	}
	LocaleSetDefault(LocaleROOT)
	LocaleSetDefaultCategory(LocaleCategoryFORMAT, LocaleUS)
	zone := TimeZoneGetDefault()
	t.Cleanup(func() { TimeZoneSetDefault(zone) })
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	if got := NewSimpleDateFormat("yyyy-MM-dd").Format(NewDate(0)); got != "1970-01-01" {
		t.Fatal(got)
	}
	if got := DateFormatGetDateTimeInstance(DateFormatMEDIUM, DateFormatMEDIUM).Format(NewDate(0)); got != "Jan 1, 1970, 12:00:00\u202fAM" {
		t.Fatal(got)
	}
	LocaleSetDefault(LocaleENGLISH)
	if LocaleGetDefaultCategory(LocaleCategoryFORMAT) != LocaleENGLISH || LocaleGetDefaultCategory(LocaleCategoryDISPLAY) != LocaleENGLISH {
		t.Fatal("general setter did not reset both categories")
	}
}

func TestLocaleCategoryNullAndConcurrentDefaults(t *testing.T) {
	preserveLocaleCategories(t)
	func() {
		defer func() {
			if p := recover(); p == nil || !CaughtAs(p, "NullPointerException") {
				t.Fatalf("null category: %v", p)
			}
		}()
		LocaleGetDefaultCategory(nil)
	}()
	LocaleSetDefault(LocaleUS)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				LocaleSetDefaultCategory(LocaleCategoryFORMAT, LocaleENGLISH)
				LocaleSetDefaultCategory(LocaleCategoryDISPLAY, LocaleUS)
				if LocaleGetDefault() != LocaleUS {
					t.Error("category write altered general default")
				}
				_ = LocaleGetDefaultCategory(LocaleCategoryFORMAT)
				_ = LocaleGetDefaultCategory(LocaleCategoryDISPLAY)
			}
		}()
	}
	workers.Wait()
}

func TestLocaleCategoryNominalEnumIdentity(t *testing.T) {
	if EnumName(LocaleCategoryFORMAT) != "FORMAT" || EnumOrdinal(LocaleCategoryFORMAT) != 1 || EnumOrdinal(LocaleCategoryDISPLAY) != 0 || !ClassLiteral("java.util.Locale$Category").IsEnum() {
		t.Fatal("category lost Java enum metadata")
	}
	values := NewReferenceArrayOf[*LocaleCategory](1, "java.util.Locale$Category")
	ReferenceArraySet(values, 0, LocaleCategoryFORMAT)
	if ReferenceArrayGet[*LocaleCategory](values, 0, "java.util.Locale$Category") != LocaleCategoryFORMAT {
		t.Fatal("category array changed constant identity")
	}
}
