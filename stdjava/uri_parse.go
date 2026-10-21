package stdjava

// This parser follows java.net.URI's JDK21 character classes and UTF16 index
// transitions. It deliberately does not delegate Java validation to net/url.
type javaURIParser struct {
	input     *JavaString
	value     *JavaURI
	ipv6Bytes int
}

type uriCharacterClass struct{ low, high uint64 }

func (class uriCharacterClass) matches(c uint16) bool {
	if c == 0 || c >= 128 {
		return false
	}
	if c < 64 {
		return class.low&(uint64(1)<<c) != 0
	}
	return class.high&(uint64(1)<<(c-64)) != 0
}
func uriClass(chars string, escaped bool) uriCharacterClass {
	class := uriCharacterClass{}
	for _, c := range chars {
		if c < 64 {
			class.low |= uint64(1) << c
		} else {
			class.high |= uint64(1) << (c - 64)
		}
	}
	if escaped {
		class.low |= 1
	}
	return class
}

var (
	uriAlpha         = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", false)
	uriDigits        = uriClass("0123456789", false)
	uriAlphanum      = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", false)
	uriHex           = uriClass("0123456789abcdefABCDEF", false)
	uriScheme        = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+.-", false)
	uriURIC          = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'();/?:@&=+$,[]", true)
	uriPath          = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'():@&=+$,;/", true)
	uriUserInfo      = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'();:&=+$,", true)
	uriRegName       = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()$,;:@&=+", true)
	uriServer        = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'();:&=+$,@[]", true)
	uriServerPercent = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'();:&=+$,@[]%", true)
	uriScope         = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.", false)
	uriHostname      = uriClass("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-", false)
	uriIPv4          = uriClass("0123456789.", false)
)

func uriSpaceChar(c uint16) bool {
	return c == 0x20 || c == 0xa0 || c == 0x1680 || c >= 0x2000 && c <= 0x200a || c == 0x2028 || c == 0x2029 || c == 0x202f || c == 0x205f || c == 0x3000
}
func (p *javaURIParser) fail(reason string, index int) {
	panic(NewJavaURISyntaxException(p.input, uriRuntimeText(reason), int32(index)))
}
func (p *javaURIParser) expected(reason string, index int) { p.fail("Expected "+reason, index) }
func (p *javaURIParser) at(start, end int, c uint16) bool {
	return start < end && p.input.units[start] == c
}
func (p *javaURIParser) atDoubleColon(start, end int) bool {
	return p.at(start, end, ':') && p.at(start+1, end, ':')
}
func (p *javaURIParser) text(start, end int) *JavaString {
	return NewJavaStringUTF16(p.input.units[start:end])
}
func (p *javaURIParser) scanStop(start, end int, chars string) int {
	for start < end {
		found := false
		for _, c := range chars {
			if p.input.units[start] == uint16(c) {
				found = true
				break
			}
		}
		if found {
			break
		}
		start++
	}
	return start
}
func (p *javaURIParser) scanErrorStop(start, end int, errors, stops string) int {
	for start < end {
		c := p.input.units[start]
		for _, e := range errors {
			if c == uint16(e) {
				return -1
			}
		}
		for _, s := range stops {
			if c == uint16(s) {
				return start
			}
		}
		start++
	}
	return start
}
func (p *javaURIParser) scanClass(start, end int, class uriCharacterClass) int {
	for start < end {
		c := p.input.units[start]
		if class.matches(c) {
			start++
			continue
		}
		if class.low&1 != 0 {
			if c == '%' {
				if start+3 <= end && uriHex.matches(p.input.units[start+1]) && uriHex.matches(p.input.units[start+2]) {
					start += 3
					continue
				}
				p.fail("Malformed escape pair", start)
			}
			if c > 128 && !uriSpaceChar(c) && c > 31 && (c < 127 || c > 159) {
				start++
				continue
			}
		}
		break
	}
	return start
}
func (p *javaURIParser) check(start, end int, class uriCharacterClass, what string) {
	if q := p.scanClass(start, end, class); q < end {
		p.fail("Illegal character in "+what, q)
	}
}

func (p *javaURIParser) parse() {
	n := len(p.input.units)
	current := p.scanErrorStop(0, n, "/?#", ":")
	if current >= 0 && p.at(current, n, ':') {
		if current == 0 {
			p.expected("scheme name", 0)
		}
		p.check(0, 1, uriAlpha, "scheme name")
		p.check(1, current, uriScheme, "scheme name")
		p.value.scheme = p.text(0, current)
		current++
		if p.at(current, n, '/') {
			current = p.hierarchical(current, n)
		} else {
			end := p.scanStop(current, n, "#")
			if end <= current {
				p.expected("scheme-specific part", current)
			}
			p.check(current, end, uriURIC, "opaque part")
			current = end
		}
	} else {
		current = p.hierarchical(0, n)
	}
	if p.at(current, n, '#') {
		p.check(current+1, n, uriURIC, "fragment")
		p.value.fragment = p.text(current+1, n)
		current = n
	}
	if current < n {
		p.fail("end of URI", current)
	}
}
func (p *javaURIParser) hierarchical(start, n int) int {
	current := start
	if p.at(current, n, '/') && p.at(current+1, n, '/') {
		current += 2
		end := p.scanStop(current, n, "/?#")
		if end > current {
			current = p.authority(current, end)
		} else if end == n {
			p.expected("authority", current)
		}
	}
	end := p.scanStop(current, n, "?#")
	p.check(current, end, uriPath, "path")
	p.value.path = p.text(current, end)
	current = end
	if p.at(current, n, '?') {
		current++
		end = p.scanStop(current, n, "#")
		p.check(current, end, uriURIC, "query")
		p.value.query = p.text(current, end)
		current = end
	}
	return current
}
func (p *javaURIParser) tryServer(start, end int) (position int, failure *URISyntaxException) {
	defer func() {
		if observed := recover(); observed != nil {
			if syntax, ok := observed.(URISyntaxException); ok {
				failure = &syntax
				position = start
			} else {
				panic(observed)
			}
		}
	}()
	position = p.server(start, end)
	if position < end {
		p.expected("end of authority", position)
	}
	return
}
func (p *javaURIParser) authority(start, end int) int {
	class := uriServer
	if p.scanStop(start, end, "]") > start {
		class = uriServerPercent
	}
	server := p.scanClass(start, end, class) == end
	regEnd := p.scanClass(start, end, uriRegName)
	registry := regEnd == end
	if registry && !server {
		p.value.authority = p.text(start, end)
		return end
	}
	position := start
	var failure *URISyntaxException
	if server {
		position, failure = p.tryServer(start, end)
		if failure != nil {
			p.value.userInfo = nil
			p.value.host = nil
			p.value.port = -1
		}
	}
	if position < end {
		if !registry {
			if failure != nil {
				panic(*failure)
			}
			if server {
				p.fail("Illegal character in authority", position)
			} else {
				p.fail("Illegal character in authority", regEnd)
			}
		}
	}
	p.value.authority = p.text(start, end)
	return end
}
func (p *javaURIParser) server(start, end int) int {
	current := start
	q := p.scanErrorStop(current, end, "/?#", "@")
	if q >= current && p.at(q, end, '@') {
		p.check(current, q, uriUserInfo, "user info")
		p.value.userInfo = p.text(current, q)
		current = q + 1
	}
	if p.at(current, end, '[') {
		current++
		q = p.scanErrorStop(current, end, "/?#", "]")
		if q > current && p.at(q, end, ']') {
			scope := p.scanStop(current, q, "%")
			if scope > current {
				p.ipv6(current, scope)
				if scope+1 == q {
					p.fail("scope id expected", -1)
				}
				p.check(scope+1, q, uriScope, "scope id")
			} else {
				p.ipv6(current, q)
			}
			p.value.host = p.text(current-1, q+1)
			current = q + 1
		} else {
			p.expected("closing bracket for IPv6 address", q)
		}
	} else {
		q = p.tryIPv4(current, end)
		if q <= current {
			q = p.hostname(current, end)
		}
		current = q
	}
	if p.at(current, end, ':') {
		current++
		q = p.scanStop(current, end, "/")
		if q > current {
			p.check(current, q, uriDigits, "port number")
			p.value.port = p.port(current, q)
			current = q
		}
	}
	if current < end {
		p.expected("port number", current)
	}
	return current
}
func (p *javaURIParser) port(start, end int) (port int32) {
	defer func() {
		if observed := recover(); observed != nil {
			if _, ok := observed.(NumberFormatException); ok {
				p.fail("Malformed port number", start)
			}
			panic(observed)
		}
	}()
	return sqlParseIntegerUnits(p.input.units, start, end)
}
func (p *javaURIParser) scanByte(start, end int) int {
	q := p.scanClass(start, end, uriDigits)
	if q <= start {
		return q
	}
	if sqlParseIntegerUnits(p.input.units, start, q) > 255 {
		return start
	}
	return q
}
func (p *javaURIParser) scanIPv4(start, end int, strict bool) int {
	limit := p.scanClass(start, end, uriIPv4)
	if limit <= start || strict && limit != end {
		return -1
	}
	current := start
	for i := 0; i < 4; i++ {
		q := p.scanByte(current, limit)
		if q <= current {
			p.fail("Malformed IPv4 address", q)
		}
		current = q
		if i < 3 {
			if !p.at(current, limit, '.') {
				p.fail("Malformed IPv4 address", current)
			}
			current++
		}
	}
	if current < limit {
		p.fail("Malformed IPv4 address", current)
	}
	return current
}
func (p *javaURIParser) tryIPv4(start, end int) (position int) {
	defer func() {
		if observed := recover(); observed != nil {
			switch observed.(type) {
			case URISyntaxException, NumberFormatException:
				position = -1
			default:
				panic(observed)
			}
		}
	}()
	position = p.scanIPv4(start, end, false)
	if position > start && position < end && !p.at(position, end, ':') {
		position = -1
	}
	if position > start {
		p.value.host = p.text(start, position)
	}
	return
}
func (p *javaURIParser) takeIPv4(start, end int, expected string) int {
	q := p.scanIPv4(start, end, true)
	if q <= start {
		p.expected(expected, start)
	}
	return q
}
func (p *javaURIParser) hostname(start, end int) int {
	current, last := start, -1
	for {
		q := p.scanClass(current, end, uriAlphanum)
		if q <= current {
			break
		}
		last = current
		current = q
		q = p.scanClass(current, end, uriHostname)
		if q > current {
			if p.input.units[q-1] == '-' {
				p.fail("Illegal character in hostname", q-1)
			}
			current = q
		}
		if !p.at(current, end, '.') {
			break
		}
		current++
		if current >= end {
			break
		}
	}
	if current < end && !p.at(current, end, ':') {
		p.fail("Illegal character in hostname", current)
	}
	if last < 0 {
		p.expected("hostname", start)
	}
	if last > start && !uriAlpha.matches(p.input.units[last]) {
		p.fail("Illegal character in hostname", last)
	}
	p.value.host = p.text(start, current)
	return current
}
func (p *javaURIParser) ipv6(start, end int) int {
	current := start
	q := p.hexSequence(current, end)
	compressed := false
	if q > current {
		current = q
		if p.atDoubleColon(current, end) {
			compressed = true
			current = p.hexPost(current+2, end)
		} else if p.at(current, end, ':') {
			current = p.takeIPv4(current+1, end, "IPv4 address")
			p.ipv6Bytes += 4
		}
	} else if p.atDoubleColon(current, end) {
		compressed = true
		current = p.hexPost(current+2, end)
	}
	if current < end {
		p.fail("Malformed IPv6 address", start)
	}
	if p.ipv6Bytes > 16 {
		p.fail("IPv6 address too long", start)
	}
	if !compressed && p.ipv6Bytes < 16 {
		p.fail("IPv6 address too short", start)
	}
	if compressed && p.ipv6Bytes == 16 {
		p.fail("Malformed IPv6 address", start)
	}
	return current
}
func (p *javaURIParser) hexPost(start, end int) int {
	if start == end {
		return start
	}
	current := start
	q := p.hexSequence(current, end)
	if q > current {
		current = q
		if p.at(current, end, ':') {
			current = p.takeIPv4(current+1, end, "hex digits or IPv4 address")
			p.ipv6Bytes += 4
		}
	} else {
		current = p.takeIPv4(current, end, "hex digits or IPv4 address")
		p.ipv6Bytes += 4
	}
	return current
}
func (p *javaURIParser) hexSequence(start, end int) int {
	current := start
	q := p.scanClass(current, end, uriHex)
	if q <= current || p.at(q, end, '.') {
		return -1
	}
	if q > current+4 {
		p.fail("IPv6 hexadecimal digit sequence too long", current)
	}
	p.ipv6Bytes += 2
	current = q
	for current < end {
		if !p.at(current, end, ':') || p.at(current+1, end, ':') {
			break
		}
		current++
		q = p.scanClass(current, end, uriHex)
		if q <= current {
			p.expected("digits for an IPv6 address", current)
		}
		if p.at(q, end, '.') {
			current--
			break
		}
		if q > current+4 {
			p.fail("IPv6 hexadecimal digit sequence too long", current)
		}
		p.ipv6Bytes += 2
		current = q
	}
	return current
}
