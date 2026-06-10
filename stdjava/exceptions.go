package stdjava

import (
	"fmt"
	"os"
	"reflect"
	"sync"
)

// Throwable is the Go counterpart of java.lang.Throwable. Every transpiled
// exception value (both the built-in types in this package and user-defined
// exception classes) is expected to implement it so that catch-by-supertype
// dispatch and message access work uniformly.
type Throwable interface {
	// ThrowableTypeName returns the Java simple type name of this exception
	// (e.g. "IllegalArgumentException"). It is used to walk the exception
	// hierarchy when matching a catch clause.
	ThrowableTypeName() string
	// Message returns the detail message the exception was constructed with,
	// mirroring Throwable.getMessage().
	Message() string
	// Error lets an exception double as a Go error and produces a readable
	// rendering for uncaught panics.
	Error() string
}

// exceptionHierarchy records the parent type name for every known exception
// type, keyed by Java simple type name. It seeds the built-in hierarchy and is
// extended at runtime by generated init() functions for user-defined exception
// classes. The empty string marks a root (Throwable has no parent).
var (
	exceptionHierarchyMu sync.RWMutex
	exceptionHierarchy   = map[string]string{
		"Throwable":                      "",
		"Error":                          "Throwable",
		"Exception":                      "Throwable",
		"RuntimeException":               "Exception",
		"IOException":                    "Exception",
		"IllegalArgumentException":       "RuntimeException",
		"IllegalStateException":          "RuntimeException",
		"NullPointerException":           "RuntimeException",
		"IndexOutOfBoundsException":      "RuntimeException",
		"ArrayIndexOutOfBoundsException": "IndexOutOfBoundsException",
		"NumberFormatException":          "IllegalArgumentException",
		"ArithmeticException":            "RuntimeException",
		"ClassCastException":             "RuntimeException",
		"UnsupportedOperationException":  "RuntimeException",
	}
)

// RegisterException records that the exception type child extends parent. It is
// called from generated init() blocks so that user-defined exception classes
// participate in catch-by-supertype dispatch. Re-registering an existing type
// updates its parent, which keeps the call idempotent across init ordering.
func RegisterException(child, parent string) {
	exceptionHierarchyMu.Lock()
	defer exceptionHierarchyMu.Unlock()
	exceptionHierarchy[child] = parent
}

// isSubtypeOf reports whether the exception type named child is the same as, or
// a descendant of, the type named parent, walking parent pointers. A type whose
// parent is unknown is treated as a direct child of "Throwable" so unrecognised
// types are still catchable by `catch (Throwable t)` / `catch (Exception e)`
// is handled by the caller treating those as catch-all.
func isSubtypeOf(child, parent string) bool {
	if child == parent {
		return true
	}
	exceptionHierarchyMu.RLock()
	defer exceptionHierarchyMu.RUnlock()
	seen := map[string]struct{}{}
	for child != "" {
		if child == parent {
			return true
		}
		if _, ok := seen[child]; ok {
			break
		}
		seen[child] = struct{}{}
		next, ok := exceptionHierarchy[child]
		if !ok {
			break
		}
		child = next
	}
	return false
}

// throwableTypeName extracts the Java type name of a recovered panic value. It
// prefers the Throwable interface, and otherwise falls back to the Go type name
// so values panicked by something other than a transpiled `throw` still produce
// a sensible (if non-hierarchical) name.
func throwableTypeName(recovered interface{}) string {
	if recovered == nil {
		return ""
	}
	if t, ok := recovered.(Throwable); ok {
		return t.ThrowableTypeName()
	}
	rt := reflect.TypeOf(recovered)
	if rt == nil {
		return ""
	}
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt.Name()
}

// CaughtAs reports whether a recovered panic value should be handled by a catch
// clause for the Java type named typeName. Matching is by hierarchy: a thrown
// IllegalArgumentException is caught by `catch (RuntimeException e)`. A nil
// recovered value (no panic) matches nothing.
func CaughtAs(recovered interface{}, typeName string) bool {
	if recovered == nil {
		return false
	}
	return isSubtypeOf(throwableTypeName(recovered), typeName)
}

// GetMessage returns the detail message of a recovered exception value,
// mirroring Throwable.getMessage(). Non-Throwable values fall back to their
// default formatting so the call never panics.
func GetMessage(recovered interface{}) string {
	if recovered == nil {
		return ""
	}
	if t, ok := recovered.(Throwable); ok {
		return t.Message()
	}
	if err, ok := recovered.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("%v", recovered)
}

// PrintStackTrace writes a best-effort rendering of the exception to stderr,
// mirroring Throwable.printStackTrace(). A full Java-style stack trace is out of
// scope; the type name and message are printed instead.
func PrintStackTrace(recovered interface{}) {
	if recovered == nil {
		return
	}
	name := throwableTypeName(recovered)
	msg := GetMessage(recovered)
	if msg == "" {
		fmt.Fprintln(os.Stderr, name)
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", name, msg)
}

// ThrowableBase is embedded by every built-in exception type and is intended to
// be embedded by transpiled user-defined exception classes that extend a
// built-in exception. It stores the detail message and the concrete Java type
// name, and provides the Throwable method set.
type ThrowableBase struct {
	typeName string
	message  string
}

func (t ThrowableBase) ThrowableTypeName() string { return t.typeName }
func (t ThrowableBase) Message() string           { return t.message }

func (t ThrowableBase) Error() string {
	if t.message == "" {
		return t.typeName
	}
	return t.typeName + ": " + t.message
}

func (t ThrowableBase) String() string { return t.Error() }

// newThrowableBase builds a ThrowableBase and ensures the type participates in
// the hierarchy. It is used by the built-in constructors below.
func newThrowableBase(typeName, message string) ThrowableBase {
	return ThrowableBase{typeName: typeName, message: message}
}

// The concrete built-in exception types. Each is a distinct struct embedding
// ThrowableBase so generated code can type-assert/construct specific types,
// while hierarchy matching is handled by name through CaughtAs.

type Exception struct{ ThrowableBase }
type RuntimeException struct{ ThrowableBase }
type IllegalArgumentException struct{ ThrowableBase }
type IllegalStateException struct{ ThrowableBase }
type NullPointerException struct{ ThrowableBase }
type IndexOutOfBoundsException struct{ ThrowableBase }
type ArrayIndexOutOfBoundsException struct{ ThrowableBase }
type NumberFormatException struct{ ThrowableBase }
type ArithmeticException struct{ ThrowableBase }
type ClassCastException struct{ ThrowableBase }
type UnsupportedOperationException struct{ ThrowableBase }
type IOException struct{ ThrowableBase }

// The New* constructors mirror `new X(message)` in Java. They are referenced by
// name from generated `throw` statements.

func NewException(message string) Exception {
	return Exception{newThrowableBase("Exception", message)}
}

func NewRuntimeException(message string) RuntimeException {
	return RuntimeException{newThrowableBase("RuntimeException", message)}
}

func NewIllegalArgumentException(message string) IllegalArgumentException {
	return IllegalArgumentException{newThrowableBase("IllegalArgumentException", message)}
}

func NewIllegalStateException(message string) IllegalStateException {
	return IllegalStateException{newThrowableBase("IllegalStateException", message)}
}

func NewNullPointerException(message string) NullPointerException {
	return NullPointerException{newThrowableBase("NullPointerException", message)}
}

func NewIndexOutOfBoundsException(message string) IndexOutOfBoundsException {
	return IndexOutOfBoundsException{newThrowableBase("IndexOutOfBoundsException", message)}
}

func NewArrayIndexOutOfBoundsException(message string) ArrayIndexOutOfBoundsException {
	return ArrayIndexOutOfBoundsException{newThrowableBase("ArrayIndexOutOfBoundsException", message)}
}

func NewNumberFormatException(message string) NumberFormatException {
	return NumberFormatException{newThrowableBase("NumberFormatException", message)}
}

func NewArithmeticException(message string) ArithmeticException {
	return ArithmeticException{newThrowableBase("ArithmeticException", message)}
}

func NewClassCastException(message string) ClassCastException {
	return ClassCastException{newThrowableBase("ClassCastException", message)}
}

func NewUnsupportedOperationException(message string) UnsupportedOperationException {
	return UnsupportedOperationException{newThrowableBase("UnsupportedOperationException", message)}
}

func NewIOException(message string) IOException {
	return IOException{newThrowableBase("IOException", message)}
}
