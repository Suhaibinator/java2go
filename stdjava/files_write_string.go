package stdjava

import (
	"fmt"
	"os"
)

// FilesWriteStringExecution accepts the Charset and OpenOption overloads while
// retaining the caller's execution for a source-defined CharSequence.toString.
func FilesWriteStringExecution(execution *Execution, path, content any, arguments ...any) *JavaPath {
	ReferenceRequireNonNull(path)
	ReferenceRequireNonNull(content)
	if execution == nil {
		execution = NewExecution()
	}
	charset := UTF_8
	if len(arguments) > 0 {
		if selected, ok := arguments[0].(*Charset); ok {
			ReferenceRequireNonNull(selected)
			charset = selected
			arguments = arguments[1:]
		}
	}
	text := JavaStringValueOfExecution(execution, content)
	ReferenceRequireNonNull(text)
	for index := 0; index < len(text.units); index++ {
		unit := text.units[index]
		width := 1
		malformed := false
		if unit >= 0xd800 && unit <= 0xdbff {
			if index+1 < len(text.units) && text.units[index+1] >= 0xdc00 && text.units[index+1] <= 0xdfff {
				width = 2
			} else {
				malformed = true
			}
		} else if unit >= 0xdc00 && unit <= 0xdfff {
			malformed = true
		}
		if malformed {
			name := "MalformedInputException"
			if charset == UTF_8 || charset == ISO_8859_1 {
				name = "UnmappableCharacterException"
			}
			panic(newThrowableBase(name, "Input length = 1"))
		}
		if charset == US_ASCII && unit > 127 {
			panic(newThrowableBase("UnmappableCharacterException", fmt.Sprintf("Input length = %d", width)))
		}
		if charset == ISO_8859_1 && unit > 255 {
			panic(newThrowableBase("UnmappableCharacterException", "Input length = 1"))
		}
		index += width - 1
	}
	data := unsignedBytes(JavaStringGetBytes(text, charset).Elements)
	flags := os.O_WRONLY
	options := []StandardOpenOption{}
	var flatten func(any)
	flatten = func(option any) {
		ReferenceRequireNonNull(option)
		switch value := option.(type) {
		case *ReferenceArray:
			for _, item := range value.elements {
				flatten(item)
			}
		case []OpenOption:
			for _, item := range value {
				flatten(item)
			}
		case StandardOpenOption:
			options = append(options, value)
		default:
			panic(NewUnsupportedOperationException("unsupported writeString option"))
		}
	}
	for _, option := range arguments {
		flatten(option)
	}
	if len(options) == 0 {
		flags |= os.O_CREATE | os.O_TRUNC
	}
	appendMode, truncate, deleteOnClose := false, false, false
	for _, option := range options {
		switch option {
		case StandardOpenOptionWrite:
		case StandardOpenOptionAppend:
			flags |= os.O_APPEND
			appendMode = true
		case StandardOpenOptionCreate:
			flags |= os.O_CREATE
		case StandardOpenOptionCreateNew:
			flags |= os.O_CREATE | os.O_EXCL
		case StandardOpenOptionTruncateExisting:
			flags |= os.O_TRUNC
			truncate = true
		case StandardOpenOptionSync, StandardOpenOptionDsync:
			flags |= os.O_SYNC
		case StandardOpenOptionDeleteOnClose:
			deleteOnClose = true
		case StandardOpenOptionSparse:
		case StandardOpenOptionRead:
			panic(NewIllegalArgumentException("READ not allowed"))
		default:
			panic(NewUnsupportedOperationException("unsupported writeString option"))
		}
	}
	if appendMode && truncate {
		panic(NewIllegalArgumentException("APPEND + TRUNCATE_EXISTING not allowed"))
	}
	target := ioPathOf(path)
	file, err := os.OpenFile(target, flags, 0666)
	if err != nil {
		throwIOException(err)
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	var removeErr error
	if deleteOnClose {
		removeErr = os.Remove(target)
	}
	if writeErr != nil {
		throwIOException(writeErr)
	}
	if closeErr != nil {
		throwIOException(closeErr)
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		throwIOException(removeErr)
	}
	if original, ok := path.(*JavaPath); ok {
		return original
	}
	return &JavaPath{path: target}
}
func init() {
	RegisterException("CharacterCodingException", "IOException")
	RegisterException("UnmappableCharacterException", "CharacterCodingException")
}
