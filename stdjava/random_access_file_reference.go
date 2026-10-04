package stdjava

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// NewRandomAccessFileStringExecution is the canonical String constructor. The
// File overload needs its own source getPath dispatch and is not admitted here.
func NewRandomAccessFileStringExecution(execution *Execution, path, mode *JavaString) *RandomAccessFile {
	requireExecution(execution)
	var file *JavaFile
	if path != nil {
		file = NewJavaFileReference(path)
	}
	flags := randomAccessFileReferenceMode(mode)
	if file == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	for _, unit := range file.pathText.units {
		if unit == 0 {
			panic(newJavaThrowableBase("FileNotFoundException", JavaStringLiteralUTF16([]uint16{'I', 'n', 'v', 'a', 'l', 'i', 'd', ' ', 'f', 'i', 'l', 'e', ' ', 'p', 'a', 't', 'h'})))
		}
	}
	opened, err := os.OpenFile(file.path, flags, 0666)
	if err != nil {
		randomAccessFileReferenceFailure(file.pathText, err)
	}
	result := randomAccessFileReferenceFinishOpen(file, opened)
	result.state.writable = randomAccessFileWritableFromOpenFlags(flags)
	return result
}

func randomAccessFileReferenceMode(mode *JavaString) int {
	if mode == nil {
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8("Cannot invoke \"String.equals(Object)\" because \"mode\" is null")))
	}
	if mode.Equals(JavaStringLiteralUTF16([]uint16{'r'})) {
		return os.O_RDONLY
	}
	if mode.Equals(JavaStringLiteralUTF16([]uint16{'r', 'w'})) {
		return os.O_RDWR | os.O_CREATE
	}
	if mode.Equals(JavaStringLiteralUTF16([]uint16{'r', 'w', 's'})) || mode.Equals(JavaStringLiteralUTF16([]uint16{'r', 'w', 'd'})) {
		// O_SYNC supplies both data and metadata durability. For rwd this is a
		// stronger minimum guarantee; exact O_DSYNC flag/performance is not claimed.
		return os.O_RDWR | os.O_CREATE | os.O_SYNC
	}
	prefix := JavaStringFromHostUTF8("Illegal mode \"")
	suffix := JavaStringFromHostUTF8("\" must be one of \"r\", \"rw\", \"rws\", or \"rwd\"")
	units := make([]uint16, 0, len(prefix.units)+len(mode.units)+len(suffix.units))
	units = append(units, prefix.units...)
	units = append(units, mode.units...)
	units = append(units, suffix.units...)
	panic(NewJavaIllegalArgumentExceptionMessage(NewJavaStringUTF16(units)))
}

// The opened descriptor remains owned here until its state and channel can be
// published together. Failed stat or a directory rejection closes it promptly.
func randomAccessFileReferenceFinishOpen(path *JavaFile, opened *os.File) *RandomAccessFile {
	published := false
	defer func() {
		if !published {
			_ = opened.Close()
		}
	}()
	info, err := opened.Stat()
	if err != nil {
		randomAccessFileReferenceFailure(path.pathText, err)
	}
	if info.IsDir() {
		randomAccessFileReferenceFailure(path.pathText, errors.New("is a directory"))
	}
	state := &randomAccessState{file: opened}
	result := &RandomAccessFile{state: state, channel: &FileChannel{state: state}}
	published = true
	return result
}

func randomAccessFileReferenceFailure(path *JavaString, err error) {
	var failure *fs.PathError
	if errors.As(err, &failure) {
		err = failure.Err
	}
	reason := err.Error()
	if len(reason) != 0 {
		reason = strings.ToUpper(reason[:1]) + reason[1:]
	}
	suffix := JavaStringFromHostUTF8(" (" + reason + ")")
	units := make([]uint16, 0, len(path.units)+len(suffix.units))
	units = append(units, path.units...)
	units = append(units, suffix.units...)
	panic(newJavaThrowableBase("FileNotFoundException", NewJavaStringUTF16(units)))
}
