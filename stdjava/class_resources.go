package stdjava

import (
	"io/fs"
	"strings"
	"sync"
)

var classResourceRegistry struct {
	sync.RWMutex
	roots []fs.FS
}

// RegisterClassResources registers an immutable classpath root. Generated
// launchers import their embedded resource package before invoking Java main.
func RegisterClassResources(root fs.FS) {
	classResourceRegistry.Lock()
	defer classResourceRegistry.Unlock()
	classResourceRegistry.roots = append(classResourceRegistry.roots, root)
}
func (class *Class) GetResourceAsStream(name string) InputStream {
	ReferenceRequireNonNull(class)
	StringRequireNonNull(name)
	if strings.HasPrefix(name, "/") {
		name = strings.TrimPrefix(name, "/")
	} else if i := strings.LastIndex(class.GetName(), "."); i >= 0 {
		name = strings.ReplaceAll(class.GetName()[:i], ".", "/") + "/" + name
	}
	if !fs.ValidPath(name) {
		return nil
	}
	classResourceRegistry.RLock()
	defer classResourceRegistry.RUnlock()
	for _, root := range classResourceRegistry.roots {
		if data, err := fs.ReadFile(root, name); err == nil {
			return NewByteArrayInputStream(signedByteArray(data))
		}
	}
	return nil
}
