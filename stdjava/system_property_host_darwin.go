//go:build darwin

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
	case "amd64":
		arch = "x86_64"
	}
	values := map[string]string{"os.name": "Mac OS X", "os.arch": arch}
	version, err := unix.Sysctl("kern.osproductversion")
	if err == nil {
		values["os.version"] = version
	}
	return values, err
}
