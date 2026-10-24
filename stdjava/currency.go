package stdjava

import (
	"sync"
	"time"
)

// JavaCurrency preserves the canonical identity and winning String reference
// of java.util.Currency. Locale operations and serialization are separate APIs.
type JavaCurrency struct{ code *JavaString }

func init()                                     { RegisterJavaType("java.util.Currency", ObjectTypeID, SerializableTypeID) }
func (*JavaCurrency) JavaDynamicTypeID() TypeID { return "java.util.Currency" }

type currencySpecialCase struct {
	cutover                                          int64
	oldCode, newCode                                 string
	oldFraction, newFraction, oldNumeric, newNumeric int32
}
type currencyOtherCode struct {
	code              string
	fraction, numeric int32
}

// currencyCodeValidAt follows the pinned JDK String lookup: simple main entry,
// ordered old/new special cases, then other codes. It does not select a country.
func currencyCodeValidAt(units []uint16, nowMillis int64) bool {
	if len(units) != 3 || units[0] < 'A' || units[0] > 'Z' || units[1] < 'A' || units[1] > 'Z' {
		return false
	}
	entry := currencyMainTable[int(units[0]-'A')*26+int(units[1]-'A')]
	if entry&0x200 == 0 && entry != 0x7f && int32(units[2])-int32('A') == int32(entry&31) {
		return true
	}
	// Every public-data code is ASCII. This check preserves exact UTF16 String
	// equality rather than narrowing a Unicode or surrogate unit into a byte.
	if units[2] > 127 {
		return false
	}
	code := string([]byte{byte(units[0]), byte(units[1]), byte(units[2])})
	for _, special := range currencySpecialCases {
		if special.oldCode == code && (special.cutover == 1<<63-1 || nowMillis < special.cutover) {
			return true
		}
		if special.newCode == code && nowMillis >= special.cutover {
			return true
		}
	}
	for _, other := range currencyOtherCodes {
		if other.code == code {
			return true
		}
	}
	return false
}

type currencyInstanceCache struct {
	mu     sync.RWMutex
	values map[string]*JavaCurrency
}

var canonicalCurrencies currencyInstanceCache

func currencyUTF16Key(units []uint16) string {
	bytes := make([]byte, len(units)*2)
	for i, unit := range units {
		bytes[2*i], bytes[2*i+1] = byte(unit>>8), byte(unit)
	}
	return string(bytes)
}

func (cache *currencyInstanceCache) get(text *JavaString, clock func() int64) *JavaCurrency {
	if text == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	key := currencyUTF16Key(text.units)
	cache.mu.RLock()
	previous := cache.values[key]
	cache.mu.RUnlock()
	if previous != nil {
		return previous
	}
	if !currencyCodeValidAt(text.units, clock()) {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	candidate := &JavaCurrency{code: text}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if previous := cache.values[key]; previous != nil {
		return previous
	}
	if cache.values == nil {
		cache.values = make(map[string]*JavaCurrency)
	}
	cache.values[key] = candidate
	return candidate
}

// CurrencyGetInstanceJavaString samples the platform wall clock on a cache
// miss. The cache precedes date validation and retains its first winning input.
func CurrencyGetInstanceJavaString(text *JavaString) *JavaCurrency {
	return canonicalCurrencies.get(text, func() int64 { return time.Now().UnixMilli() })
}

func (currency *JavaCurrency) GetCurrencyCode() *JavaString {
	if currency == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return currency.code
}

func (currency *JavaCurrency) StringJava2goExecution(_ *Execution) *JavaString {
	return currency.GetCurrencyCode()
}
