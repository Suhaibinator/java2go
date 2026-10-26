package stdjava

import (
	"io/fs"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// GetResourceAsStreamReference keeps the canonical nullable name at the Class
// boundary. The registered roots retain the existing classpath search order.
func (class *Class) GetResourceAsStreamReference(name *JavaString) InputStream {
	if class == nil || name == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	units := name.units
	if len(units) > 0 && units[0] == '/' {
		units = units[1:]
	} else if index := strings.LastIndex(class.GetName(), "."); index >= 0 {
		prefix := strings.ReplaceAll(class.GetName()[:index], ".", "/") + "/"
		units = append(utf16.Encode([]rune(prefix)), units...)
	}
	path := string(utf16.Decode(units))
	if !fs.ValidPath(path) {
		return nil
	}
	classResourceRegistry.RLock()
	defer classResourceRegistry.RUnlock()
	for _, root := range classResourceRegistry.roots {
		data, err := fs.ReadFile(root, path)
		if err != nil {
			continue
		}
		// JDK21's exploded file classpath encodes each UTF16 code unit in
		// ParseUtil.encodePath, then its file URL handler strictly decodes UTF8.
		// Supplementary pairs therefore fail when the resource URL is opened.
		// Check the general encoding contract only after locating a resource.
		if !resourceFileURLUTF8Valid(units) {
			panic(NewJavaIllegalArgumentExceptionMessage(JavaStringFromHostUTF8("Error decoding percent encoded characters")))
		}
		return NewByteArrayInputStream(signedByteArray(data))
	}
	return nil
}

func resourceFileURLUTF8Valid(units []uint16) bool {
	encoded := make([]byte, 0, len(units)*3)
	for _, unit := range units {
		switch {
		case unit <= 0x7f:
			encoded = append(encoded, byte(unit))
		case unit <= 0x7ff:
			encoded = append(encoded, byte(0xc0|unit>>6), byte(0x80|unit&0x3f))
		default:
			encoded = append(encoded, byte(0xe0|unit>>12), byte(0x80|(unit>>6)&0x3f), byte(0x80|unit&0x3f))
		}
	}
	return utf8.Valid(encoded)
}
