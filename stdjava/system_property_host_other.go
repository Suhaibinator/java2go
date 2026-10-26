//go:build !darwin && !linux

package stdjava

import "fmt"

func systemPropertyHostIdentity() (map[string]string, error) {
	return nil, fmt.Errorf("native OS property profile is not available on this target")
}
