//go:build !darwin && !linux

package stdjava

import "os"

// The native NIO path surface models Unix paths; the exact access(F_OK)
// contract is qualified on Darwin and Linux. Keep other targets buildable
// using the previous existence approximation, without claiming equivalent
// provider, error, permission or attribute semantics there.
func filesDirectoryCheckAccess(path string) error {
	_, err := os.Stat(path)
	return err
}

// Outside the native Unix qualification, keep the historical host error
// approximation without referring to Unix-only errno constants.
func filesDirectoryExistsError(err error) bool  { return os.IsExist(err) }
func filesDirectoryMissingError(err error) bool { return os.IsNotExist(err) }
func filesDirectoryErrorDetails(err error) (string, string) {
	class := "FileSystemException"
	switch {
	case os.IsExist(err):
		return "FileAlreadyExistsException", ""
	case os.IsNotExist(err):
		return "NoSuchFileException", ""
	case os.IsPermission(err):
		return "AccessDeniedException", ""
	}
	reason := err.Error()
	if len(reason) > 0 && reason[0] >= 'a' && reason[0] <= 'z' {
		reason = string(reason[0]-('a'-'A')) + reason[1:]
	}
	return class, reason
}
