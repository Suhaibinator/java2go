package stdjava

import "sort"

// These services use the JDK21 Unicode15 case data. They do not pass UTF16
// through a Go string, where an isolated surrogate would lose its identity.
func JavaStringToUpperCase(value *JavaString, locales ...*Locale) *JavaString {
	return javaStringCaseReference(value, true, locales)
}
func JavaStringToLowerCase(value *JavaString, locales ...*Locale) *JavaString {
	return javaStringCaseReference(value, false, locales)
}

type javaCase15Range struct {
	lo, hi rune
	kind   uint8
}
type javaCase15Mapping struct {
	codepoint, lower, upper rune
	fullUpper               []uint16
}
type javaCase15CCC struct {
	codepoint rune
	class     uint8
}

func javaCase15Category(cp rune) uint8 {
	i := sort.Search(len(javaCase15Categories), func(i int) bool { return javaCase15Categories[i].hi >= cp })
	if i < len(javaCase15Categories) && javaCase15Categories[i].lo <= cp {
		return javaCase15Categories[i].kind
	}
	return 0
}
func javaCase15Lookup(cp rune) javaCase15Mapping {
	i := sort.Search(len(javaCase15Mappings), func(i int) bool { return javaCase15Mappings[i].codepoint >= cp })
	if i < len(javaCase15Mappings) && javaCase15Mappings[i].codepoint == cp {
		return javaCase15Mappings[i]
	}
	return javaCase15Mapping{codepoint: cp, lower: cp, upper: cp}
}
func javaCase15CombiningClass(cp rune) uint8 {
	i := sort.Search(len(javaCase15CombiningClasses), func(i int) bool { return javaCase15CombiningClasses[i].codepoint >= cp })
	if i < len(javaCase15CombiningClasses) && javaCase15CombiningClasses[i].codepoint == cp {
		return javaCase15CombiningClasses[i].class
	}
	return 0
}
func javaCaseUTF16Points(units []uint16) []rune {
	points := make([]rune, 0, len(units))
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xd800 && u <= 0xdbff && i+1 < len(units) && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			points = append(points, 0x10000+(rune(u-0xd800)<<10)+rune(units[i+1]-0xdc00))
			i++
		} else {
			points = append(points, rune(u))
		}
	}
	return points
}
func javaCaseAppendPoint(out []uint16, cp rune) []uint16 {
	if cp <= 0xffff {
		return append(out, uint16(cp))
	}
	cp -= 0x10000
	return append(out, uint16(0xd800+(cp>>10)), uint16(0xdc00+(cp&0x3ff)))
}
func javaStringCaseReference(value *JavaString, upper bool, locales []*Locale) *JavaString {
	ReferenceRequireNonNull(value)
	var locale *Locale
	if len(locales) == 0 {
		locale = LocaleGetDefault()
	} else if len(locales) == 1 {
		locale = locales[0]
	} else {
		panic(NewIllegalArgumentException("String casing accepts zero or one locale"))
	}
	ReferenceRequireNonNull(locale)
	points := javaCaseUTF16Points(value.units)
	// JDK String first scans the simple mapping (uppercase uses its expansion
	// marker). No allocation occurs until this scan finds a mapping. An expansion
	// marker can require a new object even when its full result has equal units.
	changed := false
	for _, cp := range points {
		mapping := javaCase15Lookup(cp)
		if (upper && mapping.upper != cp) || (!upper && mapping.lower != cp) {
			changed = true
			break
		}
	}
	if !changed {
		return value
	}
	base, _ := locale.tag.Base()
	lang := base.String()
	var words []javaCaseWordSpan
	if !upper {
		for _, cp := range points {
			if cp == 0x03a3 {
				words = javaCase15Words(points)
				break
			}
		}
	}
	out := make([]uint16, 0, len(value.units))
	for index, cp := range points {
		if upper {
			if lang == "lt" && cp == 0x0307 && javaCaseAfterSoftDotted(points, index) {
				continue
			}
			if (lang == "tr" || lang == "az") && cp == 0x0069 {
				out = append(out, 0x0130)
				continue
			}
			mapping := javaCase15Lookup(cp)
			if mapping.upper < 0 {
				out = append(out, mapping.fullUpper...)
			} else {
				out = javaCaseAppendPoint(out, mapping.upper)
			}
			continue
		}
		if cp == 0x03a3 {
			mapped := rune(0x03c3)
			if javaCaseFinalCased(points, index, lang, words) {
				mapped = 0x03c2
			}
			out = javaCaseAppendPoint(out, mapped)
			continue
		}
		if cp == 0x0130 {
			out = append(out, 0x0069)
			if lang != "tr" && lang != "az" {
				out = append(out, 0x0307)
			}
			continue
		}
		if lang == "lt" {
			if (cp == 0x0049 || cp == 0x004a || cp == 0x012e) && javaCaseMoreAbove(points, index) {
				out = javaCaseAppendPoint(out, javaCase15Lookup(cp).lower)
				out = append(out, 0x0307)
				continue
			}
			switch cp {
			case 0x00cc:
				out = append(out, 0x0069, 0x0307, 0x0300)
				continue
			case 0x00cd:
				out = append(out, 0x0069, 0x0307, 0x0301)
				continue
			case 0x0128:
				out = append(out, 0x0069, 0x0307, 0x0303)
				continue
			}
		}
		if lang == "tr" || lang == "az" {
			if cp == 0x0307 && javaCaseAfterI(points, index) {
				continue
			}
			if cp == 0x0049 && !javaCaseBeforeDot(points, index) {
				out = append(out, 0x0131)
				continue
			}
		}
		out = javaCaseAppendPoint(out, javaCase15Lookup(cp).lower)
	}
	return NewJavaStringUTF16(out)
}

// The contextual conditions and Soft_Dotted set follow JDK21's
// ConditionalSpecialCasing, evaluated against the original code points.
func javaCaseMoreAbove(points []rune, index int) bool {
	for _, cp := range points[index+1:] {
		class := javaCase15CombiningClass(cp)
		if class == 230 {
			return true
		}
		if class == 0 {
			return false
		}
	}
	return false
}
func javaCaseBeforeDot(points []rune, index int) bool {
	for _, cp := range points[index+1:] {
		if cp == 0x0307 {
			return true
		}
		class := javaCase15CombiningClass(cp)
		if class == 0 || class == 230 {
			return false
		}
	}
	return false
}
func javaCaseAfterI(points []rune, index int) bool {
	for i := index - 1; i >= 0; i-- {
		if points[i] == 0x0049 {
			return true
		}
		class := javaCase15CombiningClass(points[i])
		if class == 0 || class == 230 {
			return false
		}
	}
	return false
}
func javaCaseAfterSoftDotted(points []rune, index int) bool {
	for i := index - 1; i >= 0; i-- {
		switch points[i] {
		case 0x0069, 0x006a, 0x012f, 0x0268, 0x0456, 0x0458, 0x1d62, 0x1e2d, 0x1ecb, 0x2071:
			return true
		}
		class := javaCase15CombiningClass(points[i])
		if class == 0 || class == 230 {
			return false
		}
	}
	return false
}
func javaCase15Cased(cp rune) bool {
	kind := javaCase15Category(cp)
	if kind == 1 || kind == 2 || kind == 3 {
		return true
	}
	// This is the contributory set used by JDK ConditionalSpecialCasing.
	return (cp >= 0x02b0 && cp <= 0x02b8) || (cp >= 0x02c0 && cp <= 0x02c1) ||
		(cp >= 0x02e0 && cp <= 0x02e4) || cp == 0x0345 || cp == 0x037a ||
		(cp >= 0x1d2c && cp <= 0x1d61) || (cp >= 0x2160 && cp <= 0x217f) ||
		(cp >= 0x24b6 && cp <= 0x24e9)
}

type javaCaseWordSpan struct{ start, end int }

func javaCaseWordLetter(cp rune) bool {
	kind := javaCase15Category(cp)
	if (kind < 1 || kind > 5) && kind != 8 {
		return false
	}
	// The JDK word rules group these scripts separately from letter/number words.
	switch {
	case cp == 0x3005 || (cp >= 0x4e00 && cp <= 0x9fa5) || (cp >= 0xf900 && cp <= 0xfa2d) ||
		(cp >= 0x30a1 && cp <= 0x30fa) || cp == 0x30fd || cp == 0x30fe ||
		(cp >= 0x3041 && cp <= 0x3094) || cp == 0x309d || cp == 0x309e ||
		(cp >= 0x3099 && cp <= 0x309c) || cp == 0x30fb || cp == 0x30fc:
		return false
	}
	return true
}
func javaCaseWordNumber(cp rune) bool {
	kind := javaCase15Category(cp)
	return kind >= 9 && kind <= 11
}
func javaCaseEnclosing(cp rune) bool {
	kind := javaCase15Category(cp)
	return kind == 6 || kind == 7
}
func javaCaseMidWord(cp rune) bool {
	kind := javaCase15Category(cp)
	return kind == 20 || kind == 23 || cp == 0x00ad || cp == 0x2027 || cp == '"' || cp == '\'' || cp == '.'
}
func javaCaseMidNumber(cp rune) bool {
	return cp == '"' || cp == '\'' || cp == ',' || cp == 0x066b || cp == '.'
}

// Project the default JDK WordBreakRules onto cased-letter connectivity. Format
// characters are ignored, marks attach to their base, words admit one internal
// punctuation character, numbers admit their numeric punctuation, and contiguous
// words/numbers join. Prefix/suffix punctuation and separate CJK runs contain no
// cased points and cannot change Final_Sigma's before/after result.
func javaCase15Words(points []rune) []javaCaseWordSpan {
	logical := make([]int, 0, len(points))
	for index, cp := range points {
		if javaCase15Category(cp) != 16 {
			logical = append(logical, index)
		}
	}
	spans := make([]javaCaseWordSpan, 0)
	for start := 0; start < len(logical); {
		position := start
		letter, number := javaCaseWordLetter(points[logical[position]]), javaCaseWordNumber(points[logical[position]])
		if letter || number {
			for {
				position++
				for position < len(logical) && javaCaseEnclosing(points[logical[position]]) {
					position++
				}
				if position >= len(logical) {
					break
				}
				cp := points[logical[position]]
				nextLetter, nextNumber := javaCaseWordLetter(cp), javaCaseWordNumber(cp)
				if nextLetter || nextNumber {
					letter, number = nextLetter, nextNumber
					continue
				}
				if position+1 < len(logical) {
					next := points[logical[position+1]]
					if (letter && javaCaseMidWord(cp) && javaCaseWordLetter(next)) || (number && javaCaseMidNumber(cp) && javaCaseWordNumber(next)) {
						position++
						continue
					}
				}
				break
			}
		} else {
			position++
			kind := javaCase15Category(points[logical[start]])
			if !javaCaseEnclosing(points[logical[start]]) && kind != 15 && kind != 16 && kind != 13 && kind != 14 {
				for position < len(logical) && javaCaseEnclosing(points[logical[position]]) {
					position++
				}
			}
		}
		spans = append(spans, javaCaseWordSpan{logical[start], logical[position-1] + 1})
		start = position
	}
	return spans
}
func javaCaseFinalCased(points []rune, index int, lang string, spans []javaCaseWordSpan) bool {
	for _, span := range spans {
		if index < span.start || index >= span.end {
			continue
		}
		if lang == "th" {
			for _, cp := range points[span.start:span.end] {
				if cp >= 0x0e00 && cp <= 0x0e7f && javaCase15Category(cp) != 0 {
					panic(NewUnsupportedOperationException("String casing requires JDK Thai dictionary word boundaries"))
				}
			}
		}
		preceded := false
		for _, cp := range points[span.start:index] {
			preceded = preceded || javaCase15Cased(cp)
		}
		if !preceded {
			return false
		}
		for _, cp := range points[index+1 : span.end] {
			if javaCase15Cased(cp) {
				return false
			}
		}
		return true
	}
	return false
}
