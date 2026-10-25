package stdjava

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// filesCreateDirectoriesNative follows Files.createDirectories' mkdir-first
// order. A failing first mkdir only triggers ancestor discovery when it was
// not an existing non-directory. The first mkdir and ancestor probes preserve
// lexical dot components. The fallback relativize then normalizes both operands,
// as UnixPath.relativize does, while child resolution retains the found parent.
func filesCreateDirectoriesNative(target string) {
	if err := filesCreateAndCheckDirectory(target); err == nil {
		return
	} else if filesDirectoryExistsError(err) {
		panic(filesDirectoryFailure(err))
	}
	absolute := target
	if !strings.HasPrefix(absolute, "/") {
		cwd, err := os.Getwd()
		if err != nil {
			throwIOException(err)
		}
		absolute = joinJavaPath([]string{cwd, target})
	}
	dir := &JavaPath{path: absolute}
	parent := dir.GetParent()
	for parent != nil {
		err := filesDirectoryCheckAccess(parent.path)
		if err == nil {
			break
		}
		if !filesDirectoryMissingError(err) {
			panic(filesDirectoryFailure(err))
		}
		parent = parent.GetParent()
	}
	if parent == nil {
		panic(newFilesDirectoryFailure("FileSystemException", absolute, "Unable to determine if root directory exists"))
	}
	relative, err := filepath.Rel(parent.path, absolute)
	if err != nil {
		// Both paths are absolute Unix paths in this native surface.
		panic(NewIllegalArgumentException(err.Error()))
	}
	if relative == "." {
		// UnixPath's empty relative path has one empty name. Resolving it
		// leaves parent intact, but still performs createAndCheckIsDirectory.
		relative = ""
	}
	child := parent.path
	for _, name := range strings.Split(relative, "/") {
		child = joinJavaPath([]string{child, name})
		if err := filesCreateAndCheckDirectory(child); err != nil {
			panic(filesDirectoryFailure(err))
		}
	}
}

// UnixPath.getByteArrayForSysCalls maps an empty logical path to dot. Keep
// the logical path for returns and exception fields; only the OS call changes.
func filesDirectorySystemPath(path string) string {
	if path == "" {
		return "."
	}
	return path
}

func filesCreateAndCheckDirectory(path string) error {
	systemPath := filesDirectorySystemPath(path)
	err := os.Mkdir(systemPath, 0o777)
	if filesDirectoryExistsError(err) {
		// Like Files.isDirectory, this follows symbolic links and treats attribute
		// lookup failure as false, preserving the original mkdir exception.
		if info, lookupErr := os.Stat(systemPath); lookupErr == nil && info.IsDir() {
			return nil
		}
	}
	if systemPath != path {
		if failure, ok := err.(*os.PathError); ok {
			err = &os.PathError{Op: failure.Op, Path: path, Err: failure.Err}
		}
	}
	return err
}

// This runtime-owned carrier keeps NIO's file/reason fields separate. Its
// reason is null for the three errno-specific subclasses, while the virtual
// message includes the file. No host error or user callback is retained.
type filesDirectoryException struct {
	ThrowableBase
	file    *JavaString
	reason  *JavaString
	message string
}

func newFilesDirectoryFailure(class, path, reason string) filesDirectoryException {
	var reasonReference *JavaString
	if reason != "" {
		reasonReference = JavaStringFromHostUTF8(reason)
	}
	message := path
	if reason != "" {
		message += ": " + reason
	}
	return filesDirectoryException{newJavaThrowableBase(class, reasonReference), JavaStringFromHostUTF8(path), reasonReference, message}
}
func (failure filesDirectoryException) GetFile() *JavaString      { return failure.file }
func (failure filesDirectoryException) GetOtherFile() *JavaString { return nil }
func (failure filesDirectoryException) GetReason() *JavaString    { return failure.reason }
func (failure filesDirectoryException) Message() string           { return failure.message }
func (failure filesDirectoryException) Error() string {
	return failure.ThrowableTypeName() + ": " + failure.message
}
func (failure filesDirectoryException) String() string { return failure.Error() }
func (failure filesDirectoryException) GetMessageJava2goExecution(_ *Execution) *JavaString {
	return JavaStringFromHostUTF8(failure.message)
}

func filesDirectoryFailure(err error) filesDirectoryException {
	path := ""
	var pathError *os.PathError
	if errors.As(err, &pathError) {
		path = pathError.Path
		err = pathError.Err
	}
	class, reason := filesDirectoryErrorDetails(err)
	return newFilesDirectoryFailure(class, path, reason)
}
func init() {
	RegisterException("FileSystemException", "IOException")
	RegisterException("FileAlreadyExistsException", "FileSystemException")
	RegisterException("NoSuchFileException", "FileSystemException")
	RegisterException("AccessDeniedException", "FileSystemException")
	RegisterJavaThrowableMessage(filesDirectoryException{}, func(execution *Execution, receiver any) *JavaString {
		return receiver.(filesDirectoryException).GetMessageJava2goExecution(execution)
	})
}
