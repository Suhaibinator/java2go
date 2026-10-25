package stdjava

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

const url180ProviderDescriptor = "META-INF/services/java.net.spi.URLStreamHandlerProvider"

func url180ProviderRegistryScope(t *testing.T) {
	t.Helper()
	classResourceRegistry.Lock()
	previous := classResourceRegistry.roots
	classResourceRegistry.roots = nil
	classResourceRegistry.Unlock()
	t.Cleanup(func() {
		classResourceRegistry.Lock()
		classResourceRegistry.roots = previous
		classResourceRegistry.Unlock()
	})
}

func TestURL180ProviderDescriptorRequiresExplicitRefusal(t *testing.T) {
	url180ProviderRegistryScope(t)
	RegisterClassResources(fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Data: []byte("missing.Provider\n")}})
	var value *JavaURL
	failure := url180Recovered(func() { value = NewJavaURLStringExecution(NewExecution(), uriRuntimeText("http://Example.invalid/a")) })
	if !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatalf("configured URL provider descriptor must be explicitly unsupported; actual value=%T failure=%T", value, failure)
	}
}

// Provider configuration is deliberately unprovided. These assertions promise
// explicit refusal, including empty/comment descriptors, not provider parity.
type url180ProviderOpenFS func(string) (fs.File, error)

func (open url180ProviderOpenFS) Open(name string) (fs.File, error) { return open(name) }

type url180ProviderStatFS func(string) (fs.FileInfo, error)

func (stat url180ProviderStatFS) Open(string) (fs.File, error)          { return nil, fs.ErrNotExist }
func (stat url180ProviderStatFS) Stat(name string) (fs.FileInfo, error) { return stat(name) }

type url180ProviderIsError struct{ value any }

func (url180ProviderIsError) Error() string         { return "custom inspection error" }
func (failure url180ProviderIsError) Is(error) bool { panic(failure.value) }

type url180ProviderCloseFile struct{ value any }

func (*url180ProviderCloseFile) Read([]byte) (int, error)   { return 0, io.EOF }
func (*url180ProviderCloseFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrNotExist }
func (file *url180ProviderCloseFile) Close() error          { panic(file.value) }

func TestURL180ProviderConfigurationPolicy(t *testing.T) {
	for _, entry := range []struct {
		name   string
		root   fs.FS
		refuse bool
	}{
		{"absent", fstest.MapFS{"unrelated": &fstest.MapFile{Data: []byte("value")}}, false},
		{"wrapped_absent", url180ProviderOpenFS(func(string) (fs.File, error) { return nil, fmt.Errorf("inspection: %w", fs.ErrNotExist) }), false},
		{"empty", fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{}}, true},
		{"comment", fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Data: []byte("# no implementations\n")}}, true},
		{"present", fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Data: []byte("missing.Provider\n")}}, true},
		{"directory", fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Mode: fs.ModeDir | 0o555}}, true},
		{"other_error", url180ProviderOpenFS(func(string) (fs.File, error) { return nil, errors.New("uninspectable configuration") }), true},
	} {
		t.Run(entry.name, func(t *testing.T) {
			url180ProviderRegistryScope(t)
			RegisterClassResources(entry.root)
			var value *JavaURL
			failure := url180Recovered(func() { value = NewJavaURLStringExecution(NewExecution(), uriRuntimeText("http://Example.invalid/a")) })
			if entry.refuse {
				if !CaughtAs(failure, "UnsupportedOperationException") {
					t.Fatalf("unprovided configuration became %T: %v", failure, failure)
				}
			} else if failure != nil || value == nil {
				t.Fatalf("proven absent descriptor changed default handler: %T %v", failure, failure)
			}
		})
	}
}

func TestURL180ProviderLookupOrdering(t *testing.T) {
	for _, entry := range []struct {
		input   string
		calls   int
		failure string
	}{
		{"file:/tmp/a", 0, ""}, {"jar:file:/tmp/a.jar!/entry", 0, ""},
		{"http://Example.invalid/a", 1, "UnsupportedOperationException"}, {"https://Example.invalid/a", 1, "UnsupportedOperationException"},
		{"catalog-unregistered:value", 1, "UnsupportedOperationException"}, {"jar:http://Example.invalid/a.jar!/entry", 1, "UnsupportedOperationException"},
		{"relative/catalog", 0, "MalformedURLException"}, {"jar:jar:file:/tmp/a.jar!/b.jar!/entry", 0, "MalformedURLException"},
		{"ftp:value", 0, "UnsupportedOperationException"}, {"jrt:value", 0, "UnsupportedOperationException"},
	} {
		t.Run(entry.input, func(t *testing.T) {
			url180ProviderRegistryScope(t)
			calls := 0
			RegisterClassResources(url180ProviderOpenFS(func(name string) (fs.File, error) {
				calls++
				if name != url180ProviderDescriptor {
					t.Fatalf("unexpected descriptor query %q", name)
				}
				return nil, errors.New("unprovided inspection")
			}))
			failure := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), uriRuntimeText(entry.input)) })
			if calls != entry.calls {
				t.Fatalf("lookup calls=%d want%d", calls, entry.calls)
			}
			if entry.failure == "" {
				if failure != nil {
					t.Fatalf("excluded provider lookup failed: %T %v", failure, failure)
				}
			} else if !CaughtAs(failure, entry.failure) {
				t.Fatalf("lookup order failure=%T %v, want%s", failure, failure, entry.failure)
			}
		})
	}
	t.Run("null_before_lookup", func(t *testing.T) {
		url180ProviderRegistryScope(t)
		calls := 0
		RegisterClassResources(url180ProviderOpenFS(func(string) (fs.File, error) { calls++; return nil, errors.New("unprovided") }))
		failure := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), nil) })
		if calls != 0 || !CaughtAs(failure, "MalformedURLException") {
			t.Fatalf("null input ordering calls=%d failure=%T", calls, failure)
		}
	})
	t.Run("absent_before_unknown", func(t *testing.T) {
		url180ProviderRegistryScope(t)
		calls := 0
		RegisterClassResources(url180ProviderOpenFS(func(string) (fs.File, error) { calls++; return nil, fs.ErrNotExist }))
		failure := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), uriRuntimeText("catalog-unregistered:value")) })
		if calls != 1 || !CaughtAs(failure, "MalformedURLException") {
			t.Fatalf("unknown protocol ordering calls=%d failure=%T", calls, failure)
		}
	})
}

func TestURL180ProviderCallbackAbruptIdentity(t *testing.T) {
	markers := []any{NewJavaNullPointerExceptionMessage(uriRuntimeText("fs-NPE")), NewJavaIllegalArgumentExceptionMessage(uriRuntimeText("fs-IAE")), NewJavaNumberFormatException(uriRuntimeText("fs-NFE")), NewJavaMalformedURLExceptionMessage(uriRuntimeText("fs-MUE")), new(int)}
	for i, marker := range markers {
		for _, callback := range []string{"Open", "Stat", "Close", "error.Is"} {
			for _, input := range []string{"http://Example.invalid/a", "jar:http://Example.invalid/a.jar!/entry"} {
				t.Run(fmt.Sprintf("%d/%s/%s", i, callback, input), func(t *testing.T) {
					url180ProviderRegistryScope(t)
					var root fs.FS
					switch callback {
					case "Open":
						root = url180ProviderOpenFS(func(string) (fs.File, error) { panic(marker) })
					case "Stat":
						root = url180ProviderStatFS(func(string) (fs.FileInfo, error) { panic(marker) })
					case "Close":
						root = url180ProviderOpenFS(func(string) (fs.File, error) { return &url180ProviderCloseFile{marker}, nil })
					case "error.Is":
						root = url180ProviderOpenFS(func(string) (fs.File, error) { return nil, url180ProviderIsError{marker} })
					}
					RegisterClassResources(root)
					actual := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), uriRuntimeText(input)) })
					if actual != marker {
						t.Fatalf("custom callback abrupt value changed: got %T %v, want %T %v", actual, actual, marker, marker)
					}
				})
			}
		}
	}
}

func TestURL180ProviderReentrantFreshSnapshot(t *testing.T) {
	url180ProviderRegistryScope(t)
	calls := 0
	RegisterClassResources(url180ProviderOpenFS(func(name string) (fs.File, error) {
		calls++
		if name != url180ProviderDescriptor {
			t.Fatalf("unexpected path %q", name)
		}
		nested := NewJavaURLStringExecution(NewExecution(), uriRuntimeText("file:/tmp/reentrant"))
		if nested == nil {
			t.Fatal("nested default file constructor failed")
		}
		RegisterClassResources(fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Data: []byte("missing.Provider\n")}})
		return nil, fs.ErrNotExist
	}))
	value := NewJavaURLStringExecution(NewExecution(), uriRuntimeText("http://Example.invalid/first"))
	if value == nil || calls != 1 {
		t.Fatalf("first query did not retain its initial snapshot: calls=%d", calls)
	}
	failure := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), uriRuntimeText("http://Example.invalid/second")) })
	if calls != 2 || !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatalf("new query missed late registration: calls=%d failure=%T", calls, failure)
	}
}

func TestURL180ProviderRegistryRace(t *testing.T) {
	url180ProviderRegistryScope(t)
	RegisterClassResources(fstest.MapFS{})
	var workers sync.WaitGroup
	var unexpected atomic.Int32
	input := uriRuntimeText("http://Example.invalid/race")
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 32; j++ {
				failure := url180Recovered(func() { NewJavaURLStringExecution(NewExecution(), input) })
				if failure != nil && !CaughtAs(failure, "UnsupportedOperationException") {
					unexpected.Add(1)
				}
			}
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 32; i++ {
			RegisterClassResources(fstest.MapFS{url180ProviderDescriptor: &fstest.MapFile{Data: []byte("missing.Provider\n")}})
		}
	}()
	workers.Wait()
	if unexpected.Load() != 0 {
		t.Fatalf("concurrent resource snapshot had %d unexpected failures", unexpected.Load())
	}
}
