package stdjava

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

// PatternSyntaxException retains the original Java pattern and diagnostic units.
// The canonical descriptor is independent of a source class with the same name.
type PatternSyntaxException struct {
	ThrowableBase
	description, pattern *JavaString
	index                int32
}

func (p PatternSyntaxException) GetDescription() *JavaString { return p.description }
func (p PatternSyntaxException) GetPattern() *JavaString     { return p.pattern }
func (p PatternSyntaxException) GetIndex() int32             { return p.index }

// PatternSyntaxException.getMessage builds a new Java String on every call.
func (p PatternSyntaxException) javaThrowableMessage() *JavaString {
	return CopyJavaString(p.ThrowableBase.javaThrowableMessage())
}
func (p PatternSyntaxException) JavaDynamicTypeID() TypeID {
	return "java.util.regex.PatternSyntaxException"
}
func init() {
	RegisterException("PatternSyntaxException", "IllegalArgumentException")
	RegisterJavaType("java.util.regex.PatternSyntaxException", BuiltinThrowableTypeID("IllegalArgumentException"))
}
func newSplitPatternSyntax(description string, pattern *JavaString, index int) PatternSyntaxException {
	desc := NewJavaStringUTF16(utf16.Encode([]rune(description)))
	message := slices.Clone(desc.units)
	if index >= 0 {
		message = append(message, utf16.Encode([]rune(" near index "+strconv.Itoa(index)))...)
	}
	message = append(message, '\n')
	if pattern == nil {
		message = append(message, 'n', 'u', 'l', 'l')
	} else {
		message = append(message, pattern.units...)
	}
	if index >= 0 && pattern != nil && index < len(pattern.units) {
		message = append(message, '\n')
		for _, u := range pattern.units[:index] {
			if u == '\t' {
				message = append(message, '\t')
			} else {
				message = append(message, ' ')
			}
		}
		message = append(message, '^')
	}
	return PatternSyntaxException{newJavaThrowableBase("PatternSyntaxException", NewJavaStringUTF16(message)), desc, pattern, int32(index)}
}

// JavaStringSplitArray is the canonical String[] ABI of both String.split
// overloads. Matching and substring positions remain UTF16 indexes. Captures
// guide matching but are never inserted into the result. The bounded native
// matcher rejects features outside its declared grammar; it never delegates to
// Java or silently substitutes host regular-expression semantics.
func JavaStringSplitArray(text, regex *JavaString, limits ...int32) *ReferenceArray {
	ReferenceRequireNonNull(text)
	ReferenceRequireNonNull(regex)
	limit := int32(0)
	if len(limits) > 0 {
		limit = limits[0]
	}
	if len(regex.units) > 65536 {
		splitRegexUnsupported("pattern resource limit")
	}
	if delimiter, literal := stringSplitLiteralDelimiter(regex); literal {
		return stringSplitLiteralArray(text, delimiter, limit)
	}
	parser := splitRegexParser{pattern: regex}
	node := parser.expression()
	if parser.pos < len(regex.units) {
		parser.syntax("Unmatched closing ')'", parser.pos)
	}
	matcher := splitRegexMatcher{units: text.units, steps: 1000000}
	parts := []any{}
	index, search := 0, 0
	for search <= len(text.units) {
		start, end, found := matcher.find(node, search, parser.groups, parser.supplementary)
		if !found {
			break
		}
		// Advancing one char after an empty match is intentional: Matcher.find
		// advances a UTF16 unit, even through the middle of a supplementary pair.
		search = end
		if end == start {
			search++
		}
		if index == 0 && start == 0 && end == 0 {
			continue
		}
		if limit > 0 && len(parts) == int(limit)-1 {
			parts = append(parts, text.Substring(int32(index), text.Length()))
			return ReferenceArrayLiteral(StringTypeID, parts...)
		}
		parts = append(parts, text.Substring(int32(index), int32(start)))
		index = end
	}
	if index == 0 {
		return ReferenceArrayLiteral(StringTypeID, text)
	}
	parts = append(parts, text.Substring(int32(index), text.Length()))
	if limit == 0 {
		for len(parts) > 0 && parts[len(parts)-1].(*JavaString).Length() == 0 {
			parts = parts[:len(parts)-1]
		}
	}
	return ReferenceArrayLiteral(StringTypeID, parts...)
}

// Supported grammar: concatenation/alternation, numbered captures/backrefs,
// noncapturing and atomic groups, lookahead and fixed-UTF16-width lookbehind,
// greedy/reluctant/possessive repetition, BMP/supplementary literals, simple
// character classes/ranges, Java ASCII predefined classes, Unicode categories
// and selected java properties, quoted literals, linebreaks, boundaries and
// i/u/U/m/s/d flags. Unsupported extensions fail explicitly. Capture/matching
// state is invocation-local, so concurrent calls share no mutable data.
type splitRegexNode struct {
	kind                            byte
	children                        []*splitRegexNode
	literal                         []uint16
	accept                          func(rune) bool
	flags                           uint8
	group, min, max                 int
	reluctant, possessive, negative bool
	quoted                          bool
}

const (
	splitCase uint8 = 1 << iota
	splitUnicodeCase
	splitUnicodeClass
	splitMultiline
	splitDotAll
	splitUnixLines
)

type splitRegexParser struct {
	pattern            *JavaString
	pos, groups, depth int
	flags              uint8
	supplementary      bool
}

func (p *splitRegexParser) syntax(message string, index int) {
	panic(newSplitPatternSyntax(message, p.pattern, index))
}
func splitRegexUnsupported(feature string) {
	panic(NewUnsupportedOperationException("unsupported Java regular expression feature: " + feature))
}
func (p *splitRegexParser) peek() uint16 {
	if p.pos < len(p.pattern.units) {
		return p.pattern.units[p.pos]
	}
	return 0
}
func (p *splitRegexParser) expression() *splitRegexNode {
	alternatives := []*splitRegexNode{p.sequence()}
	for p.pos < len(p.pattern.units) && p.peek() == '|' {
		p.pos++
		alternatives = append(alternatives, p.sequence())
	}
	if len(alternatives) == 1 {
		return alternatives[0]
	}
	return &splitRegexNode{kind: '|', children: alternatives}
}
func (p *splitRegexParser) sequence() *splitRegexNode {
	nodes := []*splitRegexNode{}
	for p.pos < len(p.pattern.units) && p.peek() != ')' && p.peek() != '|' {
		nodes = append(nodes, p.repetition(p.atom()))
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	return &splitRegexNode{kind: 'S', children: nodes}
}
func (p *splitRegexParser) repetition(child *splitRegexNode) *splitRegexNode {
	if p.pos == len(p.pattern.units) {
		return child
	}
	min, max := 0, 0
	switch p.peek() {
	case '*':
		max = -1
		p.pos++
	case '+':
		min = 1
		max = -1
		p.pos++
	case '?':
		max = 1
		p.pos++
	case '{':
		p.pos++
		min = p.number()
		max = min
		if p.pos < len(p.pattern.units) && p.peek() == ',' {
			p.pos++
			max = -1
			if p.pos < len(p.pattern.units) && p.peek() >= '0' && p.peek() <= '9' {
				max = p.number()
			}
		}
		if p.pos == len(p.pattern.units) || p.peek() != '}' {
			p.syntax("Unclosed counted closure", p.pos)
		}
		if max >= 0 && max < min {
			p.syntax("Illegal repetition range", p.pos)
		}
		p.pos++
	default:
		return child
	}
	var prefix []*splitRegexNode
	if child.quoted {
		if len(child.children) == 0 {
			splitRegexUnsupported("quantified empty quoted literal")
		}
		// Quoting affects lexical interpretation, not grouping. A quantifier
		// following \E binds only the last quoted codepoint.
		prefix = child.children[:len(child.children)-1]
		child = child.children[len(child.children)-1]
	}
	n := &splitRegexNode{kind: '*', children: []*splitRegexNode{child}, min: min, max: max}
	if p.pos < len(p.pattern.units) {
		if p.peek() == '?' {
			n.reluctant = true
			p.pos++
		} else if p.peek() == '+' {
			n.possessive = true
			p.pos++
		}
	}
	if p.pos < len(p.pattern.units) && strings.ContainsRune("*+?{", rune(p.peek())) {
		p.syntax("Dangling meta character '"+string(rune(p.peek()))+"'", p.pos)
	}
	if len(prefix) > 0 {
		return &splitRegexNode{kind: 'S', children: append(slices.Clone(prefix), n)}
	}
	return n
}
func (p *splitRegexParser) number() int {
	start := p.pos
	n := 0
	for p.pos < len(p.pattern.units) && p.peek() >= '0' && p.peek() <= '9' {
		d := int(p.peek() - '0')
		if n > 214748364 || n == 214748364 && d > 7 {
			p.syntax("Illegal repetition range", p.pos)
		}
		n = n*10 + d
		p.pos++
	}
	if p.pos == start {
		p.syntax("Illegal repetition", p.pos)
	}
	return n
}
func (p *splitRegexParser) literalNode(units []uint16) *splitRegexNode {
	for _, u := range units {
		if u >= 0xd800 && u <= 0xdfff {
			p.supplementary = true
		}
	}
	return &splitRegexNode{kind: 'L', literal: units, flags: p.flags}
}
func (p *splitRegexParser) codePoint() []uint16 {
	u := p.pattern.units[p.pos]
	p.pos++
	if u >= 0xd800 && u <= 0xdbff && p.pos < len(p.pattern.units) && p.peek() >= 0xdc00 && p.peek() <= 0xdfff {
		v := p.peek()
		p.pos++
		return []uint16{u, v}
	}
	return []uint16{u}
}
func (p *splitRegexParser) atom() *splitRegexNode {
	start := p.pos
	u := p.peek()
	p.pos++
	switch u {
	case '(':
		p.depth++
		if p.depth > 256 {
			splitRegexUnsupported("group nesting resource limit")
		}
		defer func() { p.depth-- }()
		flags := p.flags
		kind := byte('G')
		group := 0
		negative := false
		if p.pos < len(p.pattern.units) && p.peek() == '?' {
			p.pos++
			if p.pos == len(p.pattern.units) {
				p.syntax("Unknown inline modifier", p.pos)
			}
			switch p.peek() {
			case ':':
				kind = 'S'
				p.pos++
			case '=':
				kind = 'A'
				p.pos++
			case '!':
				kind = 'A'
				negative = true
				p.pos++
			case '>':
				kind = 'T'
				p.pos++
			case '<':
				p.pos++
				if p.pos == len(p.pattern.units) || (p.peek() != '=' && p.peek() != '!') {
					splitRegexUnsupported("named captures")
				}
				kind = 'B'
				negative = p.peek() == '!'
				p.pos++
			default:
				off := false
				for p.pos < len(p.pattern.units) && p.peek() != ')' && p.peek() != ':' {
					if p.peek() == '-' {
						off = true
						p.pos++
						continue
					}
					bit := uint8(0)
					switch p.peek() {
					case 'i':
						bit = splitCase
					case 'u':
						bit = splitUnicodeCase
					case 'U':
						bit = splitUnicodeClass | splitUnicodeCase
					case 'm':
						bit = splitMultiline
					case 's':
						bit = splitDotAll
					case 'd':
						bit = splitUnixLines
					case 'x', 'c':
						splitRegexUnsupported("comments or canonical equivalence")
					default:
						p.syntax("Unknown inline modifier", p.pos)
					}
					if off {
						p.flags &^= bit
					} else {
						p.flags |= bit
					}
					p.pos++
				}
				if p.pos == len(p.pattern.units) {
					p.syntax("Unknown inline modifier", p.pos)
				}
				if p.peek() == ')' {
					p.pos++
					return &splitRegexNode{kind: 'S'}
				}
				p.pos++
				kind = 'S'
			}
		} else {
			p.groups++
			group = p.groups
		}
		child := p.expression()
		if p.pos == len(p.pattern.units) {
			p.syntax("Unclosed group", p.pos)
		}
		p.pos++
		p.flags = flags
		n := &splitRegexNode{kind: kind, children: []*splitRegexNode{child}, group: group, negative: negative}
		if kind == 'B' && splitRegexWidth(child) < 0 {
			splitRegexUnsupported("variable UTF16 width lookbehind")
		}
		return n
	case '[':
		return p.characterClass(start)
	case '.':
		return &splitRegexNode{kind: '.', flags: p.flags}
	case '^', '$':
		return &splitRegexNode{kind: byte(u), flags: p.flags}
	case '\\':
		return p.escape(false)
	case '*', '+', '?':
		p.syntax("Dangling meta character '"+string(rune(u))+"'", start)
	case '{':
		p.syntax("Illegal repetition", p.pos)
	}
	p.pos = start
	return p.literalNode(p.codePoint())
}
func (p *splitRegexParser) hex(width int) rune {
	value := rune(0)
	for i := 0; i < width; i++ {
		if p.pos == len(p.pattern.units) {
			p.syntax("Illegal hexadecimal escape sequence", p.pos)
		}
		u := p.peek()
		p.pos++
		v := -1
		if u >= '0' && u <= '9' {
			v = int(u - '0')
		} else if u >= 'a' && u <= 'f' {
			v = int(u-'a') + 10
		} else if u >= 'A' && u <= 'F' {
			v = int(u-'A') + 10
		}
		if v < 0 {
			p.syntax("Illegal hexadecimal escape sequence", p.pos-1)
		}
		value = value*16 + rune(v)
	}
	return value
}
func splitRuneUnits(value rune) []uint16 {
	if value <= 0xffff {
		return []uint16{uint16(value)}
	}
	a, b := utf16.EncodeRune(value)
	return []uint16{uint16(a), uint16(b)}
}
func (p *splitRegexParser) escape(inClass bool) *splitRegexNode {
	if p.pos == len(p.pattern.units) {
		p.syntax("Unescaped trailing backslash", p.pos)
	}
	u := p.peek()
	p.pos++
	switch u {
	case 'Q':
		nodes := []*splitRegexNode{}
		for p.pos < len(p.pattern.units) {
			if p.peek() == '\\' && p.pos+1 < len(p.pattern.units) && p.pattern.units[p.pos+1] == 'E' {
				p.pos += 2
				break
			}
			nodes = append(nodes, p.literalNode(p.codePoint()))
		}
		return &splitRegexNode{kind: 'S', children: nodes, quoted: true}
	case 'n':
		return p.literalNode([]uint16{'\n'})
	case 'r':
		return p.literalNode([]uint16{'\r'})
	case 't':
		return p.literalNode([]uint16{'\t'})
	case 'f':
		return p.literalNode([]uint16{'\f'})
	case 'a':
		return p.literalNode([]uint16{7})
	case 'e':
		return p.literalNode([]uint16{27})
	case 'x':
		if p.pos < len(p.pattern.units) && p.peek() == '{' {
			p.pos++
			start := p.pos
			for p.pos < len(p.pattern.units) && p.peek() != '}' {
				p.pos++
			}
			if p.pos == len(p.pattern.units) {
				p.syntax("Unclosed hexadecimal escape sequence", p.pos)
			}
			str := string(utf16.Decode(p.pattern.units[start:p.pos]))
			v, err := strconv.ParseUint(str, 16, 32)
			if err != nil || v > 0x10ffff {
				p.syntax("Hexadecimal codepoint is too big", p.pos)
			}
			p.pos++
			if v >= 0xd800 && v <= 0xdfff {
				splitRegexUnsupported("escaped surrogate composition")
			}
			return p.literalNode(splitRuneUnits(rune(v)))
		}
		return p.literalNode(splitRuneUnits(p.hex(2)))
	case 'u':
		value := p.hex(4)
		if value >= 0xd800 && value <= 0xdfff {
			splitRegexUnsupported("escaped surrogate composition")
		}
		return p.literalNode(splitRuneUnits(value))
	case 'd', 'D', 'w', 'W', 's', 'S':
		flags := p.flags
		lower := u | 32
		return &splitRegexNode{kind: 'C', accept: func(r rune) bool { v := splitRegexPredefined(lower, r, flags); return v == (u == lower) }}
	case 'b', 'B':
		if inClass {
			splitRegexUnsupported("boundary escape in class")
		}
		return &splitRegexNode{kind: 'b', negative: u == 'B', flags: p.flags}
	case 'A':
		return &splitRegexNode{kind: 'a'}
	case 'z':
		return &splitRegexNode{kind: 'z'}
	case 'Z':
		return &splitRegexNode{kind: 'Z', flags: p.flags}
	case 'R':
		return &splitRegexNode{kind: 'R'}
	case 'p', 'P':
		if p.pos == len(p.pattern.units) || p.peek() != '{' {
			splitRegexUnsupported("single letter Unicode property")
		}
		p.pos++
		start := p.pos
		for p.pos < len(p.pattern.units) && p.peek() != '}' {
			p.pos++
		}
		if p.pos == len(p.pattern.units) {
			p.syntax("Unclosed character family", p.pos)
		}
		name := string(utf16.Decode(p.pattern.units[start:p.pos]))
		p.pos++
		predicate := splitRegexProperty(name, p.flags)
		return &splitRegexNode{kind: 'C', accept: func(r rune) bool { return predicate(r) == (u == 'p') }}
	}
	if u >= '1' && u <= '9' {
		if inClass {
			splitRegexUnsupported("numeric escape in class")
		}
		group := int(u - '0')
		if group > p.groups {
			splitRegexUnsupported("forward capture reference")
		}
		for p.pos < len(p.pattern.units) && p.peek() >= '0' && p.peek() <= '9' {
			next := group*10 + int(p.peek()-'0')
			if next > p.groups {
				break
			}
			group = next
			p.pos++
		}
		return &splitRegexNode{kind: 'F', group: group, flags: p.flags}
	}
	if u >= 'a' && u <= 'z' || u >= 'A' && u <= 'Z' || u == '0' {
		splitRegexUnsupported("escape \\" + string(rune(u)))
	}
	return p.literalNode([]uint16{u})
}
func (p *splitRegexParser) classAtom() *splitRegexNode {
	if p.peek() == '\\' {
		p.pos++
		return p.escape(true)
	}
	if p.peek() == '[' {
		splitRegexUnsupported("nested character class")
	}
	return p.literalNode(p.codePoint())
}
func (p *splitRegexParser) characterClass(start int) *splitRegexNode {
	negative := false
	if p.pos < len(p.pattern.units) && p.peek() == '^' {
		negative = true
		p.pos++
	}
	predicates := []func(rune) bool{}
	first := true
	for p.pos < len(p.pattern.units) && (p.peek() != ']' || first) {
		if p.peek() == '&' && p.pos+1 < len(p.pattern.units) && p.pattern.units[p.pos+1] == '&' {
			splitRegexUnsupported("character class intersection")
		}
		a := p.classAtom()
		first = false
		if a.kind != 'L' && a.kind != 'C' {
			splitRegexUnsupported("class escape")
		}
		if a.kind == 'L' && len(a.literal) > 0 && p.pos+1 < len(p.pattern.units) && p.peek() == '-' && p.pattern.units[p.pos+1] != ']' {
			p.pos++
			b := p.classAtom()
			if b.kind != 'L' || len(b.literal) == 0 {
				p.syntax("Illegal character range", p.pos-1)
			}
			lo, _ := splitRegexCodePoint(a.literal, 0)
			hi, _ := splitRegexCodePoint(b.literal, 0)
			if hi < lo {
				p.syntax("Illegal character range", p.pos-1)
			}
			flags := p.flags
			predicates = append(predicates, func(r rune) bool { return splitRegexFoldRange(r, lo, hi, flags) })
		} else if a.kind == 'C' {
			predicates = append(predicates, a.accept)
		} else {
			value, _ := splitRegexCodePoint(a.literal, 0)
			flags := p.flags
			predicates = append(predicates, func(r rune) bool { return splitRegexEqualRune(value, r, flags) })
		}
	}
	if p.pos == len(p.pattern.units) {
		p.syntax("Unclosed character class", max(start, p.pos-1))
	}
	p.pos++
	return &splitRegexNode{kind: 'C', accept: func(r rune) bool {
		found := false
		for _, f := range predicates {
			if f(r) {
				found = true
				break
			}
		}
		return found != negative
	}}
}
func splitRegexPredefined(kind uint16, r rune, flags uint8) bool {
	if flags&splitUnicodeClass == 0 {
		switch kind {
		case 'd':
			return r >= '0' && r <= '9'
		case 'w':
			return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_'
		case 's':
			return r == ' ' || r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r'
		}
	}
	splitRegexUnicodeVersion(r)
	switch kind {
	case 'd':
		return unicode.Is(unicode.Nd, r)
	case 'w':
		return unicode.IsLetter(r) || unicode.Is(unicode.Nd, r) || unicode.IsMark(r) || unicode.Is(unicode.Pc, r) || r == 0x200c || r == 0x200d
	case 's':
		return unicode.IsSpace(r)
	}
	return false
}
func splitRegexUnicodeVersion(r rune) {
	if r > 127 && unicode.Version != "15.0.0" {
		splitRegexUnsupported("Unicode table version differs from JDK21")
	}
}
func splitRegexProperty(name string, flags uint8) func(rune) bool {
	predicate := splitRegexPropertyTable(name, flags)
	stable := name == "ASCII" || name == "javaISOControl" || name == "javaWhitespace" || flags&splitUnicodeClass == 0 && (name == "Lower" || name == "Upper")
	return func(r rune) bool {
		if !stable {
			splitRegexUnicodeVersion(r)
		}
		return predicate(r)
	}
}
func splitRegexPropertyTable(name string, flags uint8) func(rune) bool {
	switch name {
	case "javaLowerCase":
		if flags&splitCase != 0 {
			return splitRegexJavaCased
		}
		return func(r rune) bool { return unicode.IsLower(r) || unicode.Is(unicode.Other_Lowercase, r) }
	case "javaUpperCase":
		if flags&splitCase != 0 {
			return splitRegexJavaCased
		}
		return func(r rune) bool { return unicode.IsUpper(r) || unicode.Is(unicode.Other_Uppercase, r) }
	case "javaLetter":
		return unicode.IsLetter
	case "javaDigit":
		return func(r rune) bool { return unicode.Is(unicode.Nd, r) }
	case "javaLetterOrDigit":
		return func(r rune) bool { return unicode.IsLetter(r) || unicode.Is(unicode.Nd, r) }
	case "javaWhitespace":
		return CharIsWhitespace
	case "javaISOControl":
		return func(r rune) bool { return r <= 0x1f || r >= 0x7f && r <= 0x9f }
	case "Lower":
		return func(r rune) bool {
			if flags&splitUnicodeClass != 0 {
				if flags&splitCase != 0 {
					return splitRegexJavaCased(r)
				}
				return unicode.IsLower(r)
			}
			if flags&splitCase != 0 {
				return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
			}
			return r >= 'a' && r <= 'z'
		}
	case "Upper":
		return func(r rune) bool {
			if flags&splitUnicodeClass != 0 {
				if flags&splitCase != 0 {
					return splitRegexJavaCased(r)
				}
				return unicode.IsUpper(r)
			}
			if flags&splitCase != 0 {
				return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
			}
			return r >= 'A' && r <= 'Z'
		}
	case "ASCII":
		return func(r rune) bool { return r <= 127 }
	}
	if table, ok := unicode.Categories[name]; ok {
		// JDK21 CharPredicates.forProperty(Lu/Ll/Lt,true) uses the union
		// of all three categories, even without UNICODE_CASE.
		if flags&splitCase != 0 && (name == "Lu" || name == "Ll" || name == "Lt") {
			return func(r rune) bool {
				return unicode.Is(unicode.Lu, r) || unicode.Is(unicode.Ll, r) || unicode.Is(unicode.Lt, r)
			}
		}
		return func(r rune) bool { return unicode.Is(table, r) }
	}
	if strings.HasPrefix(name, "Is") {
		if table, ok := unicode.Scripts[strings.TrimPrefix(name, "Is")]; ok {
			return func(r rune) bool { return unicode.Is(table, r) }
		}
	}
	splitRegexUnsupported("Unicode property " + name)
	return nil
}
func splitRegexJavaCased(r rune) bool {
	return unicode.IsLower(r) || unicode.IsUpper(r) || unicode.IsTitle(r) || unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}
func splitRegexCodePoint(units []uint16, pos int) (rune, int) {
	u := units[pos]
	if u >= 0xd800 && u <= 0xdbff && pos+1 < len(units) && units[pos+1] >= 0xdc00 && units[pos+1] <= 0xdfff {
		return utf16.DecodeRune(rune(u), rune(units[pos+1])), pos + 2
	}
	return rune(u), pos + 1
}
func splitRegexEqualRune(a, b rune, flags uint8) bool {
	if a == b {
		return true
	}
	if flags&splitCase == 0 {
		return false
	}
	if flags&splitUnicodeCase != 0 {
		splitRegexUnicodeVersion(a)
		splitRegexUnicodeVersion(b)
		return unicode.ToLower(unicode.ToUpper(a)) == unicode.ToLower(unicode.ToUpper(b))
	}
	if a >= 'A' && a <= 'Z' {
		a += 32
	}
	if b >= 'A' && b <= 'Z' {
		b += 32
	}
	return a == b
}
func splitRegexFoldRange(r, lo, hi rune, flags uint8) bool {
	if r >= lo && r <= hi {
		return true
	}
	if flags&splitCase == 0 {
		return false
	}
	lower, upper := r, r
	if flags&splitUnicodeCase != 0 {
		splitRegexUnicodeVersion(r)
		lower = unicode.ToLower(r)
		upper = unicode.ToUpper(r)
	} else {
		if r >= 'A' && r <= 'Z' {
			lower += 32
		}
		if r >= 'a' && r <= 'z' {
			upper -= 32
		}
	}
	return lower >= lo && lower <= hi || upper >= lo && upper <= hi
}
func splitRegexWidth(n *splitRegexNode) int {
	switch n.kind {
	case 'L':
		return len(n.literal)
	case 'S', 'G', 'T':
		w := 0
		for _, c := range n.children {
			v := splitRegexWidth(c)
			if v < 0 {
				return -1
			}
			w += v
		}
		return w
	case '|':
		w := -1
		for _, c := range n.children {
			v := splitRegexWidth(c)
			if v < 0 || w >= 0 && v != w {
				return -1
			}
			w = v
		}
		return w
	case '*':
		if n.max == n.min {
			w := splitRegexWidth(n.children[0])
			if w >= 0 {
				return w * n.min
			}
		}
		return -1
	case 'a', 'z', 'Z', '^', '$', 'b', 'A', 'B':
		return 0
	}
	return -1
}

type splitRegexCapture struct {
	start, end int
	set        bool
}
type splitRegexState struct {
	pos      int
	captures []splitRegexCapture
}
type splitRegexMatcher struct {
	units                     []uint16
	steps, depth, repeatDepth int
}

func (m *splitRegexMatcher) spend() {
	m.steps--
	if m.steps < 0 {
		splitRegexUnsupported("native match resource limit")
	}
}
func (m *splitRegexMatcher) find(n *splitRegexNode, from, groups int, supplementary bool) (int, int, bool) {
	for i := from; i <= len(m.units); i++ {
		if supplementary && i > 0 && i < len(m.units) && m.units[i] >= 0xdc00 && m.units[i] <= 0xdfff && m.units[i-1] >= 0xd800 && m.units[i-1] <= 0xdbff {
			continue
		}
		result := m.eval(n, splitRegexState{i, make([]splitRegexCapture, groups+1)})
		if len(result) > 0 {
			return i, result[0].pos, true
		}
	}
	return 0, 0, false
}
func splitRegexLine(r rune, flags uint8) bool {
	if flags&splitUnixLines != 0 {
		return r == '\n'
	}
	return r == '\n' || r == '\r' || r == 0x85 || r == 0x2028 || r == 0x2029
}
func (m *splitRegexMatcher) eval(n *splitRegexNode, s splitRegexState) []splitRegexState {
	m.spend()
	m.depth++
	if m.depth > 2048 {
		splitRegexUnsupported("match nesting resource limit")
	}
	defer func() { m.depth-- }()
	p := s.pos
	succeed := func(pos int) []splitRegexState { s.pos = pos; return []splitRegexState{s} }
	switch n.kind {
	case 'S':
		states := []splitRegexState{s}
		for _, child := range n.children {
			next := []splitRegexState{}
			for _, v := range states {
				next = append(next, m.eval(child, v)...)
			}
			states = next
			if len(states) == 0 {
				break
			}
		}
		return states
	case '|':
		out := []splitRegexState{}
		for _, child := range n.children {
			out = append(out, m.eval(child, s)...)
		}
		return out
	case 'L':
		end := p
		for i := 0; i < len(n.literal); {
			if end >= len(m.units) {
				return nil
			}
			a, ai := splitRegexCodePoint(n.literal, i)
			b, bi := splitRegexCodePoint(m.units, end)
			if !splitRegexEqualRune(a, b, n.flags) {
				return nil
			}
			i = ai
			end = bi
		}
		return succeed(end)
	case '.', 'C':
		if p == len(m.units) {
			return nil
		}
		r, end := splitRegexCodePoint(m.units, p)
		if n.kind == '.' {
			if n.flags&splitDotAll == 0 && splitRegexLine(r, n.flags) {
				return nil
			}
		} else if !n.accept(r) {
			return nil
		}
		return succeed(end)
	case 'G':
		out := m.eval(n.children[0], s)
		for i := range out {
			out[i].captures = slices.Clone(out[i].captures)
			out[i].captures[n.group] = splitRegexCapture{p, out[i].pos, true}
		}
		return out
	case 'T':
		out := m.eval(n.children[0], s)
		if len(out) > 1 {
			out = out[:1]
		}
		return out
	case 'F':
		capture := s.captures[n.group]
		if !capture.set {
			return nil
		}
		end := p
		for i := capture.start; i < capture.end; {
			if end >= len(m.units) {
				return nil
			}
			a, ai := splitRegexCodePoint(m.units, i)
			b, bi := splitRegexCodePoint(m.units, end)
			if !splitRegexEqualRune(a, b, n.flags) {
				return nil
			}
			i = ai
			end = bi
		}
		return succeed(end)
	case '*':
		var repeat func(splitRegexState, int) []splitRegexState
		repeat = func(v splitRegexState, count int) []splitRegexState {
			m.spend()
			m.repeatDepth++
			if m.repeatDepth > 2048 {
				splitRegexUnsupported("repetition resource limit")
			}
			defer func() { m.repeatDepth-- }()
			out := []splitRegexState{}
			if n.reluctant && count >= n.min {
				out = append(out, v)
			}
			if n.max < 0 || count < n.max {
				for _, next := range m.eval(n.children[0], v) {
					if next.pos == v.pos {
						if count+1 >= n.min {
							out = append(out, next)
						} else {
							out = append(out, repeat(next, count+1)...)
						}
						continue
					}
					out = append(out, repeat(next, count+1)...)
					if n.possessive && len(out) > 0 {
						break
					}
				}
			}
			if !n.reluctant && count >= n.min {
				out = append(out, v)
			}
			return out
		}
		out := repeat(s, 0)
		if n.possessive && len(out) > 1 {
			out = out[:1]
		}
		return out
	case 'A', 'B':
		initial := s
		if n.kind == 'B' {
			initial.pos = p - splitRegexWidth(n.children[0])
			if initial.pos < 0 {
				if n.negative {
					return succeed(p)
				}
				return nil
			}
		}
		out := m.eval(n.children[0], initial)
		match := false
		var selected splitRegexState
		for _, v := range out {
			if n.kind == 'A' || v.pos == p {
				match = true
				selected = v
				break
			}
		}
		if match == n.negative {
			return nil
		}
		if !n.negative {
			s.captures = selected.captures
		}
		return succeed(p)
	case 'a':
		if p == 0 {
			return succeed(p)
		}
	case 'z':
		if p == len(m.units) {
			return succeed(p)
		}
	case '^':
		if p == 0 {
			return succeed(p)
		}
		if n.flags&splitMultiline != 0 && p < len(m.units) && splitRegexLine(rune(m.units[p-1]), n.flags) && (m.units[p-1] != '\r' || m.units[p] != '\n' || n.flags&splitUnixLines != 0) {
			return succeed(p)
		}
	case '$', 'Z':
		if p == len(m.units) {
			return succeed(p)
		}
		if !splitRegexLine(rune(m.units[p]), n.flags) {
			return nil
		}
		if p > 0 && m.units[p] == '\n' && m.units[p-1] == '\r' && n.flags&splitUnixLines == 0 {
			return nil
		}
		if n.kind == '$' && n.flags&splitMultiline != 0 {
			return succeed(p)
		}
		if p == len(m.units)-1 || n.flags&splitUnixLines == 0 && p == len(m.units)-2 && m.units[p] == '\r' && m.units[p+1] == '\n' {
			return succeed(p)
		}
	case 'b':
		left, right := false, false
		if p > 0 {
			q := p - 1
			if q > 0 && m.units[q] >= 0xdc00 && m.units[q] <= 0xdfff && m.units[q-1] >= 0xd800 && m.units[q-1] <= 0xdbff {
				q--
			}
			r, _ := splitRegexCodePoint(m.units, q)
			left = splitRegexPredefined('w', r, n.flags)
		}
		if p < len(m.units) {
			r, _ := splitRegexCodePoint(m.units, p)
			right = splitRegexPredefined('w', r, n.flags)
		}
		if (left != right) != n.negative {
			return succeed(p)
		}
	case 'R':
		if p < len(m.units) {
			u := m.units[p]
			if u == '\r' && p+1 < len(m.units) && m.units[p+1] == '\n' {
				return succeed(p + 2)
			}
			if u == '\n' || u == '\v' || u == '\f' || u == '\r' || u == 0x85 || u == 0x2028 || u == 0x2029 {
				return succeed(p + 1)
			}
		}
	default:
		panic(fmt.Sprintf("invalid native split regex node %q", n.kind))
	}
	return nil
}
