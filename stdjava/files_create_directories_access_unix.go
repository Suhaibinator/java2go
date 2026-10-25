//go:build darwin || linux

package stdjava

import (
	"errors"
	"os"
	"syscall"
)

// JDK UnixFileSystemProvider.checkAccess with no modes tests existence using
// access(F_OK), without requesting attributes or requiring file read access.
func filesDirectoryCheckAccess(path string) error {
	if err := syscall.Access(path, 0); err != nil {
		return &os.PathError{Op: "access", Path: path, Err: err}
	}
	return nil
}

func filesDirectoryExistsError(err error) bool  { return errors.Is(err, syscall.EEXIST) }
func filesDirectoryMissingError(err error) bool { return errors.Is(err, syscall.ENOENT) }
func filesDirectoryErrorDetails(err error) (string, string) {
	class := "FileSystemException"
	reason := ""
	switch {
	case filesDirectoryExistsError(err):
		class = "FileAlreadyExistsException"
	case filesDirectoryMissingError(err):
		class = "NoSuchFileException"
	case errors.Is(err, syscall.EACCES):
		class = "AccessDeniedException"
	default:
		// POSIX strerror diagnostics begin with an ASCII capital; Go's errno
		// strings begin in lower case. Preserve the remaining reason verbatim.
		reason = err.Error()
		if len(reason) > 0 && reason[0] >= 'a' && reason[0] <= 'z' {
			reason = string(reason[0]-('a'-'A')) + reason[1:]
		}
		if errors.Is(err, syscall.ELOOP) {
			reason += " or unable to access attributes of symbolic link"
		}
	}
	return class, reason
}
