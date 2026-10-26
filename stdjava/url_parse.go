package stdjava

import (
	"errors"
	"io/fs"
)

// This parser implements only String construction with the default http,
// https, file and jar handlers. All indexing and diagnostics use UTF16 units.
type javaURLParser struct {
	input     *JavaString
	execution *Execution
}

func urlUnitsIndex(units []uint16, unit uint16, from, end int) int {
	for i := from; i < end; i++ {
		if units[i] == unit {
			return i
		}
	}
	return -1
}
func urlUnitsLastIndex(units []uint16, unit uint16) int {
	for i := len(units) - 1; i >= 0; i-- {
		if units[i] == unit {
			return i
		}
	}
	return -1
}
func urlTextJoin(prefix string, units []uint16) *JavaString {
	v := uriRuntimeText(prefix).UTF16Copy()
	return NewJavaStringUTF16(append(v, units...))
}
func urlASCIIEqualFold(units []uint16, text string) bool {
	if len(units) != len(text) {
		return false
	}
	for i, c := range units {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != uint16(text[i]) {
			return false
		}
	}
	return true
}
func urlProtocolValid(units []uint16) bool {
	if len(units) == 0 {
		return false
	}
	for i, c := range units {
		kind := javaCase15Category(rune(c))
		letter := kind >= 1 && kind <= 5
		if !letter && (i == 0 || (kind != 9 && c != '.' && c != '+' && c != '-')) {
			return false
		}
	}
	return true
}
func urlProtocolName(units []uint16) string {
	// Handler names are ASCII. Non-ASCII protocols stay UTF16 in diagnostics.
	out := make([]byte, len(units))
	for i, c := range units {
		if c > 127 {
			return ""
		}
		out[i] = byte(c)
	}
	return string(out)
}

func (p *javaURLParser) parse() *JavaURL {
	units := p.input.units
	start, limit := 0, len(units)
	for limit > 0 && units[limit-1] <= ' ' {
		limit--
	}
	for start < limit && units[start] <= ' ' {
		start++
	}
	if limit-start >= 4 && urlASCIIEqualFold(units[start:start+4], "url:") {
		start += 4
	}
	var protocol *JavaString
	if start >= len(units) || units[start] != '#' {
		for i := start; i < limit && units[i] != '/'; i++ {
			if units[i] == ':' {
				candidate := JavaStringToLowerCase(NewJavaStringUTF16(units[start:i]), LocaleROOT)
				if urlProtocolValid(candidate.units) {
					protocol = candidate
					start = i + 1
				}
				break
			}
		}
	}
	if protocol == nil {
		panic(NewJavaMalformedURLExceptionMessage(urlTextJoin("no protocol: ", units)))
	}
	name := urlProtocolName(protocol.units)
	switch name {
	case "http", "https":
		urlRequireDefaultProviderConfiguration()
	case "file", "jar":
	case "ftp", "jmod", "jrt", "mailto":
		panic(NewUnsupportedOperationException("java.net.URL default handler " + name + " is not implemented"))
	default:
		urlRequireDefaultProviderConfiguration()
		panic(NewJavaMalformedURLExceptionMessage(urlTextJoin("unknown protocol: ", protocol.units)))
	}
	value := &JavaURL{protocol: protocol}
	if index := urlUnitsIndex(units, '#', start, len(units)); index >= 0 {
		value.ref = NewJavaStringUTF16(units[index+1 : limit])
		limit = index
	}
	if name == "jar" {
		p.parseJar(value, start, limit)
		return value
	}
	p.parseHierarchical(value, start, limit)
	return value
}

func (p *javaURLParser) parseJar(value *JavaURL, start, limit int) {
	units := p.input.units
	spec := units[start:limit]
	if len(spec) >= 4 && urlASCIIEqualFold(spec[:4], "jar:") {
		panic(NewJavaNullPointerExceptionMessage(uriRuntimeText("Nested JAR URLs are not supported")))
	}
	// The JDK jar handler distinguishes an absolute spec from context syntax
	// before removing URL's whitespace/url: prefix. There is no context here.
	if len(units) < 4 || !urlASCIIEqualFold(units[:4], "jar:") {
		panic(NewJavaNullPointerExceptionMessage(uriRuntimeText(`Cannot invoke "String.lastIndexOf(int)" because "ctxFile" is null`)))
	}
	bang := -1
	for i := len(spec) - 2; i >= 0; i-- {
		if spec[i] == '!' && spec[i+1] == '/' {
			bang = i
			break
		}
	}
	if bang < 0 {
		panic(NewJavaNullPointerExceptionMessage(uriRuntimeText("no !/ in spec")))
	}
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				switch failure.(type) {
				case MalformedURLException, *MalformedURLException:
					message := urlTextJoin("invalid url: ", spec).UTF16Copy()
					message = append(message, ' ', '(')
					message = append(message, JavaThrowableToStringExecution(p.execution, failure).units...)
					message = append(message, ')')
					panic(NewJavaNullPointerExceptionMessage(NewJavaStringUTF16(message)))
				default:
					panic(failure)
				}
			}
		}()
		newJavaURLStringExecution(p.execution, NewJavaStringUTF16(spec[:bang]))
	}()
	// The legacy handler setter splits at the last question mark.
	if q := urlUnitsLastIndex(spec, '?'); q >= 0 {
		value.path = NewJavaStringUTF16(spec[:q])
		value.query = NewJavaStringUTF16(spec[q+1:])
	} else {
		value.path = NewJavaStringUTF16(spec)
	}
	// Jar's handler takes ref through the end of the original spec, including
	// trailing characters that URL's general constructor trimmed.
	if value.ref != nil {
		index := urlUnitsIndex(units, '#', limit, len(units))
		value.ref = NewJavaStringUTF16(units[index+1:])
	}
}

func (p *javaURLParser) parseHierarchical(value *JavaURL, start, limit int) {
	units := p.input.units
	if start < limit {
		if q := urlUnitsIndex(units, '?', 0, len(units)); q >= 0 && q < limit {
			value.query = NewJavaStringUTF16(units[q+1 : limit])
			limit = q
		}
	}
	var host, userInfo []uint16
	unc := limit-start >= 4 && units[start] == '/' && units[start+1] == '/' && units[start+2] == '/' && units[start+3] == '/'
	if !unc && limit-start >= 2 && units[start] == '/' && units[start+1] == '/' {
		start += 2
		end := urlUnitsIndex(units, '/', start, limit)
		if end < 0 {
			end = limit
		}
		authority := units[start:end]
		value.authority = NewJavaStringUTF16(authority)
		host = authority
		if at := urlUnitsIndex(authority, '@', 0, len(authority)); at >= 0 {
			if at != urlUnitsLastIndex(authority, '@') {
				host = nil
			} else {
				userInfo = authority[:at]
				host = authority[at+1:]
			}
		}
		port := int32(-1)
		if len(host) > 0 && host[0] == '[' {
			close := urlUnitsIndex(host, ']', 0, len(host))
			if close <= 2 {
				panic(NewJavaIllegalArgumentExceptionMessage(urlTextJoin("Invalid authority field: ", authority)))
			}
			literal := host[:close+1]
			if !urlIPv6Literal(literal[1 : len(literal)-1]) {
				panic(NewJavaIllegalArgumentExceptionMessage(urlTextJoin("Invalid host: ", literal)))
			}
			if len(host) > close+1 {
				if host[close+1] != ':' {
					panic(NewJavaIllegalArgumentExceptionMessage(urlTextJoin("Invalid authority field: ", authority)))
				}
				if len(host) > close+2 {
					port = sqlParseIntegerUnits(host, close+2, len(host))
				}
			}
			host = literal
		} else if colon := urlUnitsIndex(host, ':', 0, len(host)); colon >= 0 {
			if len(host) > colon+1 {
				port = sqlParseIntegerUnits(host, colon+1, len(host))
			}
			host = host[:colon]
		}
		if port < -1 {
			panic(NewJavaIllegalArgumentExceptionMessage(ConcatJavaStrings(uriRuntimeText("Invalid port number :"), JavaStringValueOfInt(port))))
		}
		start = end
	}
	path := units[start:limit]
	if len(path) > 0 && path[0] != '/' && value.authority != nil {
		path = append([]uint16{'/'}, path...)
	}
	value.path = NewJavaStringUTF16(path)
	urlCheckHost(host)
	if userInfo != nil {
		urlCheckMasked(userInfo, 0x84008008ffffffff & ^uint64(1<<58), 0x8000000038000001, "Illegal character found in user-info: ", false)
	}
	if value.authority != nil {
		urlCheckMasked(value.authority.units, 0x84008008ffffffff & ^uint64(0x400000000000000), 0x8000000038000001 & ^uint64(0x28000001), "Illegal character found in authority: ", false)
	}
}

func urlDescribeChar(c uint16) *JavaString {
	if c < 32 || c == 127 {
		switch c {
		case '\n':
			return uriRuntimeText("LF")
		case '\r':
			return uriRuntimeText("CR")
		}
		return ConcatJavaStrings(ConcatJavaStrings(uriRuntimeText("control char (code="), JavaStringValueOfInt(int32(c))), uriRuntimeText(")"))
	}
	return NewJavaStringUTF16([]uint16{'\'', c, '\''})
}
func urlCheckMasked(units []uint16, low, high uint64, prefix string, others bool) {
	for _, c := range units {
		bad := (c < 64 && (low&(uint64(1)<<c)) != 0) || (c >= 64 && c < 128 && (high&(uint64(1)<<(c-64))) != 0)
		if others {
			switch c {
			case 8263, 8264, 8265, 8448, 8449, 8453, 8454, 10868, 65109, 65110, 65119, 65131, 65283, 65295, 65306, 65311, 65312:
				bad = true
			}
		}
		if bad {
			panic(NewJavaIllegalArgumentExceptionMessage(ConcatJavaStrings(uriRuntimeText(prefix), urlDescribeChar(c))))
		}
	}
}
func urlCheckHost(host []uint16) {
	if len(host) > 1 && host[0] == '[' && host[len(host)-1] == ']' {
		inside := host[1 : len(host)-1]
		if !urlIPv6Literal(inside) {
			panic(NewJavaIllegalArgumentExceptionMessage(uriRuntimeText("Unrecognized IPv6 address format")))
		}
		if zone := urlUnitsIndex(inside, '%', 0, len(inside)); zone >= 0 {
			urlCheckMasked(inside[zone:], 0xffffffff|uint64(1)<<47|uint64(1)<<35|uint64(1)<<63, 0x8000000000000000|uint64(1)<<28|uint64(1)<<27|uint64(1)<<29, "Illegal character found in IPv6 scoped address: ", false)
		}
	} else {
		urlCheckMasked(host, 0x84008008ffffffff|0xffffffff|uint64(1)<<47|0x5000000500000000, 0x8000000038000001|0x3800000178000000, "Illegal character found in host: ", true)
	}
}

// Validate a numeric IPv6 literal without DNS or host-machine Unicode tables.
// The optional zone is syntax here; it is never resolved to a network interface.
func urlIPv6Literal(units []uint16) bool {
	if zone := urlUnitsIndex(units, '%', 0, len(units)); zone >= 0 {
		if zone == len(units)-1 {
			return false
		}
		units = units[:zone]
	}
	if len(units) < 2 {
		return false
	}
	groups, compressed := 0, false
	i := 0
	if units[0] == ':' {
		if units[1] != ':' {
			return false
		}
		compressed = true
		i = 2
	}
	for i < len(units) {
		begin := i
		for i < len(units) && units[i] != ':' {
			i++
		}
		part := units[begin:i]
		if len(part) == 0 {
			return false
		}
		if urlUnitsIndex(part, '.', 0, len(part)) >= 0 {
			if i != len(units) || groups > 6 || len(part) > 15 {
				return false
			}
			count := 0
			pos := 0
			for pos < len(part) {
				j := pos
				value := 0
				for j < len(part) && part[j] != '.' {
					if part[j] < '0' || part[j] > '9' {
						return false
					}
					value = value*10 + int(part[j]-'0')
					if value > 255 {
						return false
					}
					j++
				}
				if j == pos {
					return false
				}
				count++
				pos = j + 1
				if j == len(part)-1 {
					return false
				}
			}
			if count != 4 {
				return false
			}
			groups += 2
			break
		}
		// JDK IPAddressUtil bounds the accumulated word, not its text length:
		// additional leading ASCII zeroes remain valid without integer overflow.
		word := 0
		for _, c := range part {
			var digit int
			switch {
			case c >= '0' && c <= '9':
				digit = int(c - '0')
			case c >= 'a' && c <= 'f':
				digit = int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				digit = int(c-'A') + 10
			default:
				return false
			}
			word = word*16 + digit
			if word > 0xffff {
				return false
			}
		}
		groups++
		if groups > 8 {
			return false
		}
		if i < len(units) {
			i++
			if i < len(units) && units[i] == ':' {
				if compressed {
					return false
				}
				compressed = true
				i++
			} else if i == len(units) {
				return false
			}
		}
	}
	return groups == 8 && !compressed || groups < 8 && compressed
}

const urlProviderServiceDescriptor = "META-INF/services/java.net.spi.URLStreamHandlerProvider"

// Presence denotes an unprovided configuration, even for empty/comment-only
// descriptors. This is a conservative service refusal, not ServiceLoader or
// provider execution. Each constructor observes a fresh registry snapshot.
type urlProviderAbrupt struct{ value any }

func urlProviderDescriptorAbsent(root fs.FS) (absent bool) {
	defer func() {
		if failure := recover(); failure != nil {
			panic(urlProviderAbrupt{failure})
		}
	}()
	_, err := fs.Stat(root, urlProviderServiceDescriptor)
	return err != nil && errors.Is(err, fs.ErrNotExist)
}

func urlRequireDefaultProviderConfiguration() {
	classResourceRegistry.RLock()
	roots := append([]fs.FS(nil), classResourceRegistry.roots...)
	classResourceRegistry.RUnlock()
	// Custom FS and error.Is callbacks run after releasing the registry lock.
	for _, root := range roots {
		if !urlProviderDescriptorAbsent(root) {
			panic(NewUnsupportedOperationException("java.net.URL handler-provider configuration is not implemented"))
		}
	}
}
