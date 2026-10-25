//go:build linux

package stdjava

import (
	"golang.org/x/sys/unix"
	"runtime"
)

func systemPropertyHostIdentity() (map[string]string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "386":
		arch = "i386"
	}
	values := map[string]string{"os.name": "Linux", "os.arch": arch}
	var identity unix.Utsname
	err := unix.Uname(&identity)
	if err == nil {
		raw := make([]byte, 0, len(identity.Release))
		for _, value := range identity.Release {
			if value == 0 {
				break
			}
			raw = append(raw, byte(value))
		}
		values["os.version"] = string(raw)
	}
	return values, err
}
