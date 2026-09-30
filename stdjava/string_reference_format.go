package stdjava

import (
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/cases"
)

type javaFormatSpecifier struct {
	index            int // 0 ordinary, -1 previous, -2 no argument, positive explicit
	flags            string
	width, precision int
	conversion       uint16
}
type javaFormatToken struct {
	literal   []uint16
	specifier *javaFormatSpecifier
}

// JavaStringFormatExecution accepts the actual Java Object[] boundary. Variable
// arity packing belongs to the caller, which knows each argument's static type.
func JavaStringFormatExecution(execution *Execution, format *JavaString, args *ReferenceArray) *JavaString {
	return JavaStringFormatLocaleExecution(execution, LocaleUS, format, args)
}

// This is the text, character, boolean, hash and integral Formatter surface.
// Floating point, dates and Formattable callbacks require their own ABI and
// report explicit unsupported capability, never a Go formatting approximation.
func JavaStringFormatLocaleExecution(execution *Execution, locale *Locale, format *JavaString, args *ReferenceArray) *JavaString {
	// Installed JDK21 Formatter.parse dereferences its String parameter s before
	// examining arguments. Preserve that service's helpful-null diagnostic and
	// canonical Throwable message/cause slots without rendering any callbacks.
	if format == nil {
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8(`Cannot invoke "String.length()" because "s" is null`)))
	}
	tokens := parseJavaFormat(format.units) // JDK parses the whole format first.
	last, ordinary := -1, -1
	var result []uint16
	for _, token := range tokens {
		if token.specifier == nil {
			result = append(result, token.literal...)
			continue
		}
		spec := token.specifier
		var value any
		if spec.index != -2 {
			index := spec.index - 1
			switch spec.index {
			case 0:
				ordinary++
				index = ordinary
			case -1:
				index = last
			}
			if index < 0 || (args != nil && index >= len(args.elements)) {
				javaFormatFailure("MissingFormatArgumentException", "Format specifier '"+spec.text()+"'")
			}
			last = index
			if args != nil {
				value = args.elements[index]
			}
		}
		result = append(result, formatJavaArgument(execution, locale, spec, value)...)
	}
	// StringBuilder.toString() constructs a String even when the content is empty.
	return NewJavaStringUTF16(result)
}

func parseJavaFormat(units []uint16) []javaFormatToken {
	var tokens []javaFormatToken
	for cursor := 0; cursor < len(units); {
		start := cursor
		for cursor < len(units) && units[cursor] != '%' {
			cursor++
		}
		if cursor > start {
			tokens = append(tokens, javaFormatToken{literal: units[start:cursor]})
		}
		if cursor == len(units) {
			break
		}
		cursor++
		if cursor == len(units) {
			javaFormatUnknownConversion([]uint16{'%'})
		}
		first := cursor
		spec := &javaFormatSpecifier{width: -1, precision: -1}
		digitStart := cursor
		for cursor < len(units) && formatDigit(units[cursor]) {
			cursor++
		}
		if cursor > digitStart && cursor < len(units) && units[cursor] == '$' {
			spec.index = parseFormatNumber(units[digitStart:cursor], "IllegalFormatArgumentIndexException")
			if spec.index <= 0 {
				javaFormatFailure("IllegalFormatArgumentIndexException", "Illegal format argument index = "+strconv.Itoa(spec.index))
			}
			cursor++
		} else {
			cursor = digitStart
		}
		for cursor < len(units) && strings.ContainsRune("-#+ 0,(<", rune(units[cursor])) {
			flag := string(rune(units[cursor]))
			if strings.Contains(spec.flags, flag) {
				javaFormatFailure("DuplicateFormatFlagsException", "Flags = '"+flag+"'")
			}
			spec.flags += flag
			cursor++
		}
		if strings.Contains(spec.flags, "<") {
			spec.index = -1
		}
		digitStart = cursor
		for cursor < len(units) && formatDigit(units[cursor]) {
			cursor++
		}
		if cursor > digitStart {
			spec.width = parseFormatNumber(units[digitStart:cursor], "IllegalFormatWidthException")
		}
		if cursor < len(units) && units[cursor] == '.' {
			cursor++
			digitStart = cursor
			for cursor < len(units) && formatDigit(units[cursor]) {
				cursor++
			}
			if cursor == digitStart {
				javaFormatUnknownConversion(units[first : first+1])
			}
			spec.precision = parseFormatNumber(units[digitStart:cursor], "IllegalFormatPrecisionException")
		}
		if cursor == len(units) {
			javaFormatUnknownConversion(units[first : first+1])
		}
		spec.conversion = units[cursor]
		cursor++
		if spec.conversion == 't' || spec.conversion == 'T' {
			panic(NewUnsupportedOperationException("String.format date/time conversion is unavailable"))
		}
		if !strings.ContainsRune("bBhHsScCdoxXeEfgGaA%n", rune(spec.conversion)) {
			javaFormatUnknownConversion([]uint16{spec.conversion})
		}
		if strings.ContainsRune("eEfgGaA", rune(spec.conversion)) {
			panic(NewUnsupportedOperationException("String.format floating-point conversion is unavailable"))
		}
		spec.validate()
		tokens = append(tokens, javaFormatToken{specifier: spec})
	}
	return tokens
}

func formatDigit(unit uint16) bool { return unit >= '0' && unit <= '9' }
func parseFormatNumber(units []uint16, failure string) int {
	bytes := make([]byte, len(units))
	for i, unit := range units {
		bytes[i] = byte(unit)
	}
	value, err := strconv.ParseInt(string(bytes), 10, 32)
	if err != nil {
		if failure == "IllegalFormatArgumentIndexException" {
			javaFormatFailure(failure, "Format argument index: (not representable as int)")
		}
		javaFormatFailure(failure, "-2147483648")
	}
	return int(value)
}
func (spec *javaFormatSpecifier) has(flag string) bool { return strings.Contains(spec.flags, flag) }
func (spec *javaFormatSpecifier) text() string {
	var text strings.Builder
	text.WriteByte('%')
	text.WriteString(spec.flagText(false))
	if spec.index > 0 {
		text.WriteString(strconv.Itoa(spec.index))
		text.WriteByte('$')
	}
	if spec.width >= 0 {
		text.WriteString(strconv.Itoa(spec.width))
	}
	if spec.precision >= 0 {
		text.WriteByte('.')
		text.WriteString(strconv.Itoa(spec.precision))
	}
	text.WriteRune(rune(spec.conversion))
	return text.String()
}
func (spec *javaFormatSpecifier) flagText(uppercase bool) string {
	var text strings.Builder
	for _, flag := range "-^#+ 0,(<" {
		if flag == '^' {
			if uppercase && spec.conversion >= 'A' && spec.conversion <= 'Z' {
				text.WriteRune(flag)
			}
		} else if spec.has(string(flag)) {
			text.WriteRune(flag)
		}
	}
	return text.String()
}
func (spec *javaFormatSpecifier) mismatch(flags string) {
	javaFormatFailure("FormatFlagsConversionMismatchException", "Conversion = "+string(rune(formatLower(spec.conversion)))+", Flags = "+flags)
}
func (spec *javaFormatSpecifier) badFlags(allowed string) {
	var bad strings.Builder
	for _, flag := range "-#+ 0,(" {
		if spec.has(string(flag)) && !strings.ContainsRune(allowed, flag) {
			bad.WriteRune(flag)
		}
	}
	if bad.Len() > 0 {
		spec.mismatch(bad.String())
	}
}
func (spec *javaFormatSpecifier) missingWidth() {
	if spec.width < 0 {
		javaFormatFailure("MissingFormatWidthException", spec.text())
	}
}
func (spec *javaFormatSpecifier) validate() {
	conversion := formatLower(spec.conversion)
	switch conversion {
	case 's', 'b', 'h':
		if conversion != 's' && spec.has("#") {
			spec.mismatch("#")
		}
		if spec.has("-") {
			spec.missingWidth()
		}
		allowed := "-"
		if conversion == 's' {
			allowed += "#"
		}
		spec.badFlags(allowed)
	case 'c':
		if spec.precision >= 0 {
			javaFormatFailure("IllegalFormatPrecisionException", strconv.Itoa(spec.precision))
		}
		spec.badFlags("-")
		if spec.has("-") {
			spec.missingWidth()
		}
	case 'd', 'o', 'x':
		if spec.has("-") || spec.has("0") {
			spec.missingWidth()
		}
		if spec.has("+") && spec.has(" ") || spec.has("-") && spec.has("0") {
			javaFormatFailure("IllegalFormatFlagsException", "Flags = '"+spec.flagText(true)+"'")
		}
		if spec.precision >= 0 {
			javaFormatFailure("IllegalFormatPrecisionException", strconv.Itoa(spec.precision))
		}
		if conversion == 'd' {
			spec.badFlags("-+ 0,(")
		} else {
			spec.badFlags("-#+ 0(")
		}
	case '%':
		spec.index = -2
		if spec.precision >= 0 {
			javaFormatFailure("IllegalFormatPrecisionException", strconv.Itoa(spec.precision))
		}
		if spec.flags != "" && spec.flags != "-" {
			javaFormatFailure("IllegalFormatFlagsException", "Flags = '"+spec.flagText(true)+"'")
		}
		if spec.has("-") {
			spec.missingWidth()
		}
	case 'n':
		spec.index = -2
		if spec.precision >= 0 {
			javaFormatFailure("IllegalFormatPrecisionException", strconv.Itoa(spec.precision))
		}
		if spec.width >= 0 {
			javaFormatFailure("IllegalFormatWidthException", strconv.Itoa(spec.width))
		}
		if spec.flags != "" {
			javaFormatFailure("IllegalFormatFlagsException", "Flags = '"+spec.flagText(true)+"'")
		}
	}
}
func formatLower(c uint16) uint16 {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func formatJavaArgument(execution *Execution, locale *Locale, spec *javaFormatSpecifier, value any) []uint16 {
	var text *JavaString
	switch formatLower(spec.conversion) {
	case 's':
		if id, known := ObjectDynamicType(value); known && (JavaTypeAssignable(id, "java.util.Formattable") || JavaTypeAssignable(id, "Formattable")) {
			panic(NewUnsupportedOperationException("String.format Formattable callback is unavailable"))
		}
		if spec.has("#") {
			spec.mismatch("#")
		}
		text = JavaStringValueOfExecution(execution, value)
	case 'b':
		truth := !javaReferenceIsNull(value)
		if boolean, ok := value.(*Boolean); ok && boolean != nil {
			truth = boolean.BooleanValue()
		}
		text = JavaStringValueOfBoolean(truth)
	case 'h':
		if javaReferenceIsNull(value) {
			text = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
		} else {
			text = javaStringFromNumericASCII(strconv.FormatUint(uint64(uint32(ObjectHashCodeExecution(execution, value))), 16), false)
		}
	case 'c':
		if javaReferenceIsNull(value) {
			text = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
		} else {
			code, ok := javaFormatCharacter(value)
			if !ok {
				javaFormatConversionFailure(spec, value)
			}
			if code < 0 || code > 0x10FFFF {
				javaFormatFailure("IllegalFormatCodePointException", "Code point = 0x"+strconv.FormatUint(uint64(uint32(code)), 16))
			}
			if code <= 0xFFFF {
				text = NewJavaStringUTF16([]uint16{uint16(code)})
			} else {
				text = NewJavaStringUTF16(utf16.Encode([]rune{rune(code)}))
			}
		}
	case 'd', 'o', 'x':
		if javaReferenceIsNull(value) {
			text = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
		} else {
			return formatJavaInteger(locale, spec, value)
		}
	case '%':
		text = NewJavaStringUTF16([]uint16{'%'})
	case 'n':
		if runtime.GOOS == "windows" {
			return []uint16{'\r', '\n'}
		}
		return []uint16{'\n'}
	}
	// A callback's null result differs from a null argument: append alone writes
	// null, but precision/case/width dereference that returned String first.
	if spec.precision >= 0 {
		length := text.Length()
		if spec.precision < int(length) {
			text = NewJavaStringUTF16(text.units[:spec.precision])
		}
	}
	if spec.conversion >= 'A' && spec.conversion <= 'Z' {
		text = javaFormatUpper(text, locale)
	}
	if spec.width >= 0 {
		return formatJavaWidth(text.unitsChecked(), spec.width, spec.has("-"))
	}
	if text == nil {
		return []uint16{'n', 'u', 'l', 'l'}
	}
	return text.units
}
func (text *JavaString) unitsChecked() []uint16 { RequireJavaString(text); return text.units }
func formatJavaWidth(units []uint16, width int, right bool) []uint16 {
	if width <= len(units) {
		return units
	}
	result := make([]uint16, width)
	for i := range result {
		result[i] = ' '
	}
	if right {
		copy(result, units)
	} else {
		copy(result[width-len(units):], units)
	}
	return result
}
func javaFormatUpper(text *JavaString, locale *Locale) *JavaString {
	RequireJavaString(text)
	if locale == nil {
		locale = LocaleUS
	}
	var result []uint16
	for start := 0; start < len(text.units); {
		end := start
		for end < len(text.units) {
			unit := text.units[end]
			if unit >= 0xD800 && unit <= 0xDBFF && end+1 < len(text.units) && text.units[end+1] >= 0xDC00 && text.units[end+1] <= 0xDFFF {
				end += 2
				continue
			}
			if unit >= 0xD800 && unit <= 0xDFFF {
				break
			}
			end++
		}
		if end > start {
			mapped := cases.Upper(locale.tag).String(string(utf16.Decode(text.units[start:end])))
			result = append(result, utf16.Encode([]rune(mapped))...)
		}
		if end < len(text.units) {
			result = append(result, text.units[end])
			end++
		}
		start = end
	}
	return NewJavaStringUTF16(result)
}
func javaFormatCharacter(value any) (int32, bool) {
	switch value := value.(type) {
	case *Character:
		return int32(value.CharValue()), true
	case *Byte:
		return int32(value.ByteValue()), true
	case *Short:
		return int32(value.ShortValue()), true
	case *Integer:
		return value.IntValue(), true
	}
	return 0, false
}
func javaFormatConversionFailure(spec *javaFormatSpecifier, value any) {
	name := "java.lang.Object"
	if id, known := ObjectDynamicType(value); known {
		name = string(id)
		if !strings.Contains(name, ".") {
			switch id {
			case StringTypeID, BooleanTypeID, ByteTypeID, ShortTypeID, CharacterTypeID, IntegerTypeID, LongTypeID, FloatTypeID, DoubleTypeID, ObjectTypeID:
				name = "java.lang." + name
			}
		}
	}
	javaFormatFailure("IllegalFormatConversionException", string(rune(formatLower(spec.conversion)))+" != "+name)
}

func formatJavaInteger(locale *Locale, spec *javaFormatSpecifier, value any) []uint16 {
	if locale != nil && locale.tag != LocaleROOT.tag && locale.tag != LocaleENGLISH.tag && locale.tag != LocaleUS.tag {
		panic(NewUnsupportedOperationException("String.format locale-specific integral formatting is unavailable"))
	}
	base := 10
	switch formatLower(spec.conversion) {
	case 'o':
		base = 8
	case 'x':
		base = 16
	}
	var signed int64
	bits := 0
	switch value := value.(type) {
	case *Byte:
		signed = int64(value.ByteValue())
		bits = 8
	case *Short:
		signed = int64(value.ShortValue())
		bits = 16
	case *Integer:
		signed = int64(value.IntValue())
		bits = 32
	case *Long:
		signed = value.LongValue()
		bits = 64
	}
	negative := false
	digits := ""
	if integer, ok := value.(*BigInteger); ok {
		negative = integer.value.Sign() < 0
		digits = new(big.Int).Abs(&integer.value).Text(base)
	} else if bits != 0 {
		if base != 10 {
			spec.badFlags("-#0")
			unsigned := uint64(signed)
			if bits < 64 {
				unsigned &= (uint64(1) << bits) - 1
			}
			digits = strconv.FormatUint(unsigned, base)
		} else {
			negative = signed < 0
			magnitude := uint64(signed)
			if negative {
				magnitude = uint64(-(signed + 1)) + 1
			}
			digits = strconv.FormatUint(magnitude, 10)
		}
	} else {
		javaFormatConversionFailure(spec, value)
	}
	if base == 10 && spec.has(",") {
		digits = javaFormatGroupedDigits(digits)
	}
	prefix, suffix := "", ""
	if negative {
		if spec.has("(") {
			prefix = "("
			suffix = ")"
		} else {
			prefix = "-"
		}
	} else if spec.has("+") {
		prefix = "+"
	} else if spec.has(" ") {
		prefix = " "
	}
	if spec.has("#") {
		if base == 8 {
			prefix += "0"
		} else if base == 16 {
			prefix += "0x"
		}
	}
	if spec.has("0") && spec.width > len(prefix)+len(digits)+len(suffix) {
		digits = strings.Repeat("0", spec.width-len(prefix)-len(digits)-len(suffix)) + digits
	}
	rendered := prefix + digits + suffix
	if spec.conversion == 'X' {
		rendered = strings.ToUpper(rendered)
	}
	units := javaStringFromNumericASCII(rendered, false).units
	return formatJavaWidth(units, spec.width, spec.has("-"))
}
func javaFormatGroupedDigits(digits string) string {
	var result strings.Builder
	for i, b := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(b)
	}
	return result.String()
}
