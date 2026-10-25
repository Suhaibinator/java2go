package stdjava

import (
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

// JavaInetAddress preserves address value equality and allocation identity.
// Resolver calls use the host's configured OS resolver; numeric literals never
// perform DNS. Resolver-provider/property overrides and interface-name scopes
// are outside this service and fail explicitly.
type JavaInetAddress struct {
	address  [16]byte
	ipv4     bool
	scope    int32
	hostname *JavaString
}
type UnknownHostException struct{ ThrowableBase }

func NewJavaUnknownHostExceptionMessage(message *JavaString) UnknownHostException {
	return UnknownHostException{newJavaThrowableBase("UnknownHostException", message)}
}
func init() {
	RegisterException("UnknownHostException", "IOException")
	RegisterJavaType("java.net.InetAddress", ObjectTypeID, SerializableTypeID)
	RegisterJavaType("java.net.Inet4Address", "java.net.InetAddress")
	RegisterJavaType("java.net.Inet6Address", "java.net.InetAddress")
}
func (value *JavaInetAddress) JavaDynamicTypeID() TypeID {
	if value.ipv4 {
		return "java.net.Inet4Address"
	}
	return "java.net.Inet6Address"
}
func requireJavaInetAddress(value *JavaInetAddress) *JavaInetAddress {
	if value == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return value
}

var javaInetLoopback = &JavaInetAddress{address: [16]byte{127, 0, 0, 1}, ipv4: true, scope: -1, hostname: uriRuntimeText("localhost")}

func JavaInetAddressGetByNameExecution(execution *Execution, host *JavaString) *JavaInetAddress {
	if host == nil || host.Length() == 0 {
		return javaInetLoopback
	}
	for _, unit := range host.units {
		if unit == 0 {
			panic(NewJavaUnknownHostExceptionMessage(uriRuntimeText("NUL character not allowed in hostname")))
		}
	}
	original := string(utf16.Decode(host.units))
	text := original
	bracketed := false
	invalid6 := func() {
		panic(NewJavaUnknownHostExceptionMessage(javaInetDiagnostic(host, ": invalid IPv6 address literal")))
	}
	if text[0] == '[' {
		if len(text) <= 2 || text[len(text)-1] != ']' {
			invalid6()
		}
		text = text[1 : len(text)-1]
		bracketed = true
	}
	first := text[0]
	if (first >= '0' && first <= '9') || (first >= 'a' && first <= 'f') || (first >= 'A' && first <= 'F') || first == ':' {
		if !bracketed {
			if address, ok := javaInetParseIPv4(text); ok {
				return &JavaInetAddress{address: address, ipv4: true, scope: -1}
			}
			if javaInetBSDParsable(text) {
				failure := NewJavaUnknownHostExceptionMessage(host)
				_ = ThrowableInitCauseDefaultExecution(execution, failure, NewJavaIllegalArgumentExceptionMessage(NewJavaStringUTF16(append(uriRuntimeText("Invalid IP address literal: ").UTF16Copy(), host.units...))))
				panic(failure)
			}
		}
		if strings.Contains(text, ":") || bracketed {
			numeric := text
			scope := int32(-1)
			namedScope := false
			if pos := strings.IndexByte(text, '%'); pos >= 0 {
				if pos == len(text)-1 {
					invalid6()
				}
				numeric = text[:pos]
				zone := text[pos+1:]
				allDigits := true
				for _, c := range zone {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if !allDigits {
					namedScope = true
				} else {
					parsed, err := strconv.ParseInt(zone, 10, 32)
					if err != nil {
						panic(NewUnsupportedOperationException("java.net.InetAddress scope overflow service is not implemented"))
					}
					scope = int32(parsed)
				}
			}
			address, ok := javaInetParseIPv6(numeric)
			if !ok {
				invalid6()
			}
			mapped := true
			for i := 0; i < 10; i++ {
				mapped = mapped && address[i] == 0
			}
			mapped = mapped && address[10] == 255 && address[11] == 255
			if mapped {
				if scope >= 0 || namedScope {
					messageHost := host
					if bracketed {
						messageHost = NewJavaStringUTF16(host.units[1 : len(host.units)-1])
					}
					message := javaInetDiagnostic(messageHost, ": invalid IPv4-mapped address")
					panic(NewJavaUnknownHostExceptionMessage(message))
				}
				return &JavaInetAddress{address: [16]byte{address[12], address[13], address[14], address[15]}, ipv4: true, scope: -1}
			}
			if namedScope {
				panic(NewUnsupportedOperationException("java.net.InetAddress interface-name scope service is not implemented"))
			}
			return &JavaInetAddress{address: address, scope: scope}
		}
	} else if bracketed {
		invalid6()
	}
	return javaInetResolve(host, original)
}

func javaInetDiagnostic(host *JavaString, suffix string) *JavaString {
	return NewJavaStringUTF16(append(host.UTF16Copy(), utf16.Encode([]rune(suffix))...))
}

func javaInetParseIPv4(text string) (address [16]byte, ok bool) {
	if len(text) == 0 || len(text) > 15 {
		return address, false
	}
	parts := strings.Split(text, ".")
	if len(parts) > 4 {
		return address, false
	}
	for i, part := range parts {
		if part == "" {
			return address, false
		}
		value := uint64(0)
		for _, c := range part {
			if c < '0' || c > '9' {
				return address, false
			}
			value = value*10 + uint64(c-'0')
		}
		limit := uint64(255)
		if i == len(parts)-1 {
			limit = uint64(1) << uint((5-len(parts))*8)
			limit--
		}
		if value > limit {
			return address, false
		}
		if i < len(parts)-1 {
			address[i] = byte(value)
		} else {
			for j := 3; j >= i; j-- {
				address[j] = byte(value)
				value >>= 8
			}
		}
	}
	return address, true
}
func javaInetBSDParsable(text string) bool {
	parts := strings.Split(text, ".")
	if len(parts) > 4 {
		return false
	}
	if len(text) == 0 || text[0] < '0' || text[0] > '9' {
		return false
	}
	for i, part := range parts {
		if part == "" {
			return false
		}
		base := 10
		digits := part
		if strings.HasPrefix(part, "0x") || strings.HasPrefix(part, "0X") {
			base = 16
			digits = part[2:]
		} else if len(part) > 1 && part[0] == '0' {
			base = 8
			digits = part[1:]
		}
		if digits == "" {
			return false
		}
		value, err := strconv.ParseUint(digits, base, 32)
		if err != nil {
			return false
		}
		limit := uint64(255)
		if i == len(parts)-1 {
			limit = (uint64(1) << uint((5-len(parts))*8)) - 1
		}
		if value > limit {
			return false
		}
	}
	return true
}
func javaInetParseIPv6(text string) (address [16]byte, ok bool) {
	if strings.Count(text, "::") > 1 {
		return address, false
	}
	if strings.Contains(text, ".") && strings.LastIndexByte(text, '.') < strings.LastIndexByte(text, ':') {
		return address, false
	}
	compressed := strings.Contains(text, "::")
	halves := strings.SplitN(text, "::", 2)
	parse := func(part string) (words []uint16, valid bool) {
		if part == "" {
			return nil, true
		}
		chunks := strings.Split(part, ":")
		for i, chunk := range chunks {
			if strings.Contains(chunk, ".") {
				if i != len(chunks)-1 || strings.Count(chunk, ".") != 3 {
					return nil, false
				}
				v4, yes := javaInetParseIPv4(chunk)
				if !yes {
					return nil, false
				}
				words = append(words, uint16(v4[0])<<8|uint16(v4[1]), uint16(v4[2])<<8|uint16(v4[3]))
				continue
			}
			if chunk == "" {
				return nil, false
			}
			value := uint64(0)
			for _, c := range chunk {
				digit := int64(-1)
				switch {
				case c >= '0' && c <= '9':
					digit = int64(c - '0')
				case c >= 'a' && c <= 'f':
					digit = int64(c - 'a' + 10)
				case c >= 'A' && c <= 'F':
					digit = int64(c - 'A' + 10)
				}
				if digit < 0 {
					return nil, false
				}
				value = value*16 + uint64(digit)
				if value > 65535 {
					return nil, false
				}
			}
			words = append(words, uint16(value))
		}
		return words, true
	}
	left, valid := parse(halves[0])
	if !valid {
		return address, false
	}
	var right []uint16
	if compressed {
		right, valid = parse(halves[1])
		if !valid || len(left)+len(right) >= 8 {
			return address, false
		}
	} else if len(left) != 8 {
		return address, false
	}
	if len(left)+len(right) > 8 {
		return address, false
	}
	for i, v := range left {
		address[i*2] = byte(v >> 8)
		address[i*2+1] = byte(v)
	}
	offset := 8 - len(right)
	for i, v := range right {
		address[(offset+i)*2] = byte(v >> 8)
		address[(offset+i)*2+1] = byte(v)
	}
	return address, true
}

func (value *JavaInetAddress) GetHostAddressJava2goExecution(_ *Execution) *JavaString {
	value = requireJavaInetAddress(value)
	var text string
	if value.ipv4 {
		text = strconv.Itoa(int(value.address[0])) + "." + strconv.Itoa(int(value.address[1])) + "." + strconv.Itoa(int(value.address[2])) + "." + strconv.Itoa(int(value.address[3]))
	} else {
		parts := make([]string, 8)
		for i := range parts {
			parts[i] = strconv.FormatUint(uint64(value.address[2*i])<<8|uint64(value.address[2*i+1]), 16)
		}
		text = strings.Join(parts, ":")
		if value.scope >= 0 {
			text += "%" + strconv.FormatInt(int64(value.scope), 10)
		}
	}
	return uriRuntimeText(text)
}
func (value *JavaInetAddress) StringJava2goExecution(execution *Execution) *JavaString {
	value = requireJavaInetAddress(value)
	var units []uint16
	if value.hostname != nil {
		units = value.hostname.UTF16Copy()
	}
	units = append(units, '/')
	units = append(units, value.GetHostAddressJava2goExecution(execution).units...)
	return NewJavaStringUTF16(units)
}
func (value *JavaInetAddress) EqualsJava2goExecution(_ *Execution, other any) bool {
	value = requireJavaInetAddress(value)
	candidate, ok := other.(*JavaInetAddress)
	return ok && candidate != nil && value.ipv4 == candidate.ipv4 && value.address == candidate.address
}
func (value *JavaInetAddress) HashCodeJava2goExecution(_ *Execution) int32 {
	value = requireJavaInetAddress(value)
	if value.ipv4 {
		return int32(uint32(value.address[0])<<24 | uint32(value.address[1])<<16 | uint32(value.address[2])<<8 | uint32(value.address[3]))
	}
	var hash int32
	for i := 0; i < 16; i += 4 {
		var word int32
		for j := 0; j < 4; j++ {
			word = (word << 8) + int32(int8(value.address[i+j]))
		}
		hash += word
	}
	return hash
}

type javaInetCacheEntry struct {
	value   *JavaInetAddress
	expires time.Time
}

var javaInetCache = struct {
	sync.Mutex
	entries map[string]javaInetCacheEntry
}{entries: map[string]javaInetCacheEntry{}}

func javaInetResolve(host *JavaString, text string) *JavaInetAddress {
	javaInetCache.Lock()
	defer javaInetCache.Unlock()
	if entry, ok := javaInetCache.entries[text]; ok && time.Now().Before(entry.expires) {
		return entry.value
	}
	addresses, err := net.LookupIP(text)
	if err != nil || len(addresses) == 0 {
		detail := ""
		if err != nil {
			detail = err.Error()
			if dns, ok := err.(*net.DNSError); ok {
				detail = dns.Err
				if dns.IsNotFound {
					switch runtime.GOOS {
					case "darwin":
						detail = "nodename nor servname provided, or not known"
					case "linux":
						detail = "Name or service not known"
					}
				}
			}
		}
		message := host
		if detail != "" {
			message = javaInetDiagnostic(host, ": "+detail)
		}
		panic(NewJavaUnknownHostExceptionMessage(message))
	}
	// Default JDK21 address selection prefers IPv4 before IPv6.
	selected := addresses[0]
	for _, address := range addresses {
		if address.To4() != nil {
			selected = address
			break
		}
	}
	value := &JavaInetAddress{scope: -1, hostname: host}
	if v4 := selected.To4(); v4 != nil {
		value.ipv4 = true
		copy(value.address[:4], v4)
	} else {
		copy(value.address[:], selected.To16())
	}
	javaInetCache.entries[text] = javaInetCacheEntry{value: value, expires: time.Now().Add(30 * time.Second)}
	return value
}
