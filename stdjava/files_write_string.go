package stdjava

import "os"

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
	text := StringValueOfExecution(execution, content)
	StringRequireNonNull(text)
	if charset == US_ASCII || charset == ISO_8859_1 {
		maximum := rune(127)
		if charset == ISO_8859_1 {
			maximum = 255
		}
		for _, r := range text {
			if r > maximum {
				panic(newThrowableBase("UnmappableCharacterException", "Input length = 1"))
			}
		}
	}
	data := unsignedBytes(StringGetBytes(text, charset).Elements)
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
