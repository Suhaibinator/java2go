package stdjava

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
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
		"Throwable":                       "",
		"Error":                           "Throwable",
		"AssertionError":                  "Error",
		"LinkageError":                    "Error",
		"ExceptionInInitializerError":     "LinkageError",
		"NoClassDefFoundError":            "LinkageError",
		"Exception":                       "Throwable",
		"RuntimeException":                "Exception",
		"IOException":                     "Exception",
		"IllegalArgumentException":        "RuntimeException",
		"IllegalStateException":           "RuntimeException",
		"IllegalMonitorStateException":    "RuntimeException",
		"NullPointerException":            "RuntimeException",
		"NegativeArraySizeException":      "RuntimeException",
		"IndexOutOfBoundsException":       "RuntimeException",
		"ArrayIndexOutOfBoundsException":  "IndexOutOfBoundsException",
		"ArrayStoreException":             "RuntimeException",
		"NumberFormatException":           "IllegalArgumentException",
		"ArithmeticException":             "RuntimeException",
		"ClassCastException":              "RuntimeException",
		"UnsupportedOperationException":   "RuntimeException",
		"NoSuchElementException":          "RuntimeException",
		"ConcurrentModificationException": "RuntimeException",
	}
)

func registerExceptionJavaType(child, parent string) {
	if child == "" {
		return
	}
	super := ObjectTypeID
	if parent != "" {
		super = TypeID(parent)
	}
	RegisterJavaType(TypeID(child), super)
}

func init() {
	for child, parent := range exceptionHierarchy {
		registerExceptionJavaType(child, parent)
	}
}

// RegisterException records that the exception type child extends parent. It is
// called from generated init() blocks so that user-defined exception classes
// participate in catch-by-supertype dispatch. Re-registering a type with the
// same parent is idempotent across init ordering.
//
// LIMITATION: the hierarchy is keyed by simple (unqualified) type name. Two
// user-defined exception classes with the same simple name in different Java
// packages collide in this map; the last registration wins for both. This is
// acceptable for the common single-package case. When a collision with a
// *different* parent is detected, a warning is printed to stderr so the
// ambiguity is at least visible rather than silently mis-dispatching.
func RegisterException(child, parent string) {
	exceptionHierarchyMu.Lock()
	if existing, ok := exceptionHierarchy[child]; ok && existing != parent {
		fmt.Fprintf(os.Stderr,
			"stdjava: exception type %q registered with conflicting parents %q and %q; "+
				"catch-by-supertype dispatch for %q may be ambiguous (simple-name collision across packages)\n",
			child, existing, parent, child)
	}
	exceptionHierarchy[child] = parent
	exceptionHierarchyMu.Unlock()
	registerExceptionJavaType(child, parent)
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
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt.Name()
}

// NormalizePanic converts a recovered panic value into the Java exception it
// corresponds to. Transpiled `throw` statements already panic with stdjava
// Throwable values, which pass through unchanged. Native Go runtime panics are
// mapped to the matching Java exception so they can be caught by the usual catch
// dispatch:
//
//	integer divide by zero  -> ArithmeticException
//	nil pointer dereference  -> NullPointerException
//	slice/array index range  -> ArrayIndexOutOfBoundsException
//	negative slice length    -> NegativeArraySizeException
//	failed type assertion    -> ClassCastException
//
// The Go runtime does not expose distinct exported types for most of these
// (they are unexported runtime.Error implementations), so the mapping inspects
// the *runtime.TypeAssertionError type and otherwise the runtime.Error message.
// A nil value (no panic) and any value that is already a Throwable are returned
// as-is. This is meant to be called at the recover boundary in generated code.
func NormalizePanic(recovered interface{}) interface{} {
	if recovered == nil {
		return nil
	}
	if _, ok := recovered.(Throwable); ok {
		return recovered
	}

	// A failed type assertion surfaces as *runtime.TypeAssertionError, which is
	// the Go analogue of a Java ClassCastException.
	if _, ok := recovered.(*runtime.TypeAssertionError); ok {
		return NewClassCastException(errorMessage(recovered))
	}

	rerr, ok := recovered.(runtime.Error)
	if !ok {
		// Any other panic value (a panicked string, a plain error, or any other
		// Go value) is wrapped as a RuntimeException. This keeps every non-typed
		// panic catchable through the normal hierarchy so `catch (Exception e)`
		// and `catch (RuntimeException e)` can be routed through CaughtAs rather
		// than treated as unconditional catch-alls, matching Java semantics where
		// catch (Exception) does not catch Error/Throwable-level throws.
		return NewRuntimeException(errorMessage(recovered))
	}

	msg := rerr.Error()
	switch {
	case strings.Contains(msg, "integer divide by zero"),
		strings.Contains(msg, "division by zero"):
		return NewArithmeticException("/ by zero")
	case strings.Contains(msg, "index out of range"),
		strings.Contains(msg, "slice bounds out of range"):
		return NewArrayIndexOutOfBoundsException(msg)
	case strings.Contains(msg, "makeslice: len out of range"):
		return NewNegativeArraySizeException(msg)
	case strings.Contains(msg, "nil pointer dereference"),
		strings.Contains(msg, "invalid memory address"):
		return NewNullPointerException(msg)
	default:
		// Some other runtime panic (e.g. a closed-channel send): a RuntimeException
		// is the closest Java analogue.
		return NewRuntimeException(msg)
	}
}

// errorMessage extracts the message from a value that implements error, falling
// back to default formatting otherwise.
func errorMessage(v interface{}) string {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("%v", v)
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

// AddSuppressed attaches suppressed to primary, mirroring the part of
// Throwable.addSuppressed used by javac's try-with-resources lowering. Values
// produced by the built-in constructors, and generated user exceptions that
// embed them, share suppression state across Go value copies.
func AddSuppressed(primary, suppressed interface{}) {
	if sameThrowableIdentity(primary, suppressed) {
		panic(NewIllegalArgumentExceptionWithCause("Self-suppression not permitted", primary))
	}
	carrier, ok := primary.(interface {
		AddSuppressedValue(interface{})
	})
	if !ok {
		return
	}
	carrier.AddSuppressedValue(suppressed)
}

// sameThrowableIdentity compares the shared state allocated for one Java
// Throwable object. Transpiled built-in exceptions are Go values, so ordinary
// interface equality is not a sound identity test: copies of one throwable
// must compare as the same Java object, while separately constructed values
// with identical type and message must remain distinct.
func sameThrowableIdentity(left, right interface{}) bool {
	leftCarrier, leftOK := left.(interface {
		ThrowableIdentity() *throwableState
	})
	rightCarrier, rightOK := right.(interface {
		ThrowableIdentity() *throwableState
	})
	if !leftOK || !rightOK {
		return false
	}
	leftIdentity := leftCarrier.ThrowableIdentity()
	return leftIdentity != nil && leftIdentity == rightCarrier.ThrowableIdentity()
}

// GetSuppressed returns a snapshot of the exceptions attached to primary, in
// the order in which resource close failures occurred.
func GetSuppressed(primary interface{}) []interface{} {
	carrier, ok := primary.(interface {
		SuppressedValues() []interface{}
	})
	if !ok {
		return []interface{}{}
	}
	return carrier.SuppressedValues()
}

// GetSuppressedArray exposes Throwable.getSuppressed through the generated
// descriptor-bearing array ABI. SuppressedValues is runtime-owned and already
// guarantees every element is a Throwable, so the trusted copy does not bypass
// any user-visible array store check.
func GetSuppressedArray(primary interface{}) *ReferenceArray {
	return SuppressedArray(GetSuppressed(primary))
}

// SuppressedArray converts the runtime snapshot to the generated Java-array
// ABI while keeping GetSuppressed's slice API available to runtime callers.
func SuppressedArray(values []interface{}) *ReferenceArray {
	array := NewReferenceArray(len(values), ThrowableTypeID)
	copy(array.elements, values)
	return array
}

// GetCause returns the cause associated with a throwable, mirroring
// Throwable.getCause(). Most constructors leave it nil; the
// IllegalArgumentException raised for self-suppression records the original
// throwable as its cause, matching the JDK implementation.
func GetCause(primary interface{}) interface{} {
	carrier, ok := primary.(interface {
		CauseValue() interface{}
	})
	if !ok {
		return nil
	}
	return carrier.CauseValue()
}

// CloseResource implements the exceptional-completion rules for one Java
// try-with-resources resource. Generated code must call it directly from a
// defer statement so recover observes a panic already propagating out of the
// resource body. A close panic becomes the primary panic only when the body
// completed normally; otherwise it is attached to the body's primary panic.
// Chaining one defer per resource naturally preserves Java's reverse close
// order and suppressed-exception ordering.
func CloseResource(closeFn func()) {
	primary := recover()
	if primary != nil {
		primary = NormalizePanic(primary)
	}

	var closePanic interface{}
	func() {
		defer func() {
			closePanic = recover()
		}()
		closeFn()
	}()

	if closePanic != nil {
		closePanic = NormalizePanic(closePanic)
		if primary == nil {
			primary = closePanic
		} else {
			AddSuppressed(primary, closePanic)
		}
	}

	if primary != nil {
		panic(primary)
	}
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
	state    *throwableState
}

type throwableState struct {
	mu         sync.Mutex
	suppressed []interface{}
	cause      interface{}
}

func (t ThrowableBase) ThrowableTypeName() string { return t.typeName }
func (t ThrowableBase) Message() string           { return t.message }

// JavaDynamicTypeID lets every built-in Throwable value participate in the
// descriptor-bearing reference-array ABI. The method is promoted through each
// concrete exception and through generated subclasses that embed one.
func (t ThrowableBase) JavaDynamicTypeID() TypeID { return TypeID(t.typeName) }

func (t ThrowableBase) Error() string {
	if t.message == "" {
		return t.typeName
	}
	return t.typeName + ": " + t.message
}

func (t ThrowableBase) String() string { return t.Error() }

// ThrowableIdentity returns the state pointer that represents this Java
// Throwable's object identity. The method is promoted through generated
// exception subclasses that embed a built-in exception type.
func (t ThrowableBase) ThrowableIdentity() *throwableState { return t.state }

// AddSuppressedValue and SuppressedValues are exported so exception classes in
// generated packages inherit the suppression carrier contract by embedding a
// built-in exception type.
func (t ThrowableBase) AddSuppressedValue(suppressed interface{}) {
	if t.state == nil {
		return
	}
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.suppressed = append(t.state.suppressed, suppressed)
}

func (t ThrowableBase) SuppressedValues() []interface{} {
	if t.state == nil {
		return []interface{}{}
	}
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	return append([]interface{}{}, t.state.suppressed...)
}

func (t ThrowableBase) CauseValue() interface{} {
	if t.state == nil {
		return nil
	}
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	return t.state.cause
}

// newThrowableBase builds a ThrowableBase and ensures the type participates in
// the hierarchy. It is used by the built-in constructors below.
func newThrowableBase(typeName, message string) ThrowableBase {
	return ThrowableBase{
		typeName: typeName,
		message:  message,
		state:    &throwableState{},
	}
}

// The concrete built-in exception types. Each is a distinct struct embedding
// ThrowableBase so generated code can type-assert/construct specific types,
// while hierarchy matching is handled by name through CaughtAs.

type Error struct{ ThrowableBase }
type AssertionError struct{ ThrowableBase }
type LinkageError struct{ ThrowableBase }
type ExceptionInInitializerError struct{ ThrowableBase }
type NoClassDefFoundError struct{ ThrowableBase }
type Exception struct{ ThrowableBase }
type RuntimeException struct{ ThrowableBase }
type IllegalArgumentException struct{ ThrowableBase }
type IllegalStateException struct{ ThrowableBase }
type IllegalMonitorStateException struct{ ThrowableBase }
type NullPointerException struct{ ThrowableBase }
type NegativeArraySizeException struct{ ThrowableBase }
type IndexOutOfBoundsException struct{ ThrowableBase }
type ArrayIndexOutOfBoundsException struct{ ThrowableBase }
type ArrayStoreException struct{ ThrowableBase }
type NumberFormatException struct{ ThrowableBase }
type ArithmeticException struct{ ThrowableBase }
type ClassCastException struct{ ThrowableBase }
type UnsupportedOperationException struct{ ThrowableBase }
type IOException struct{ ThrowableBase }
type NoSuchElementException struct{ ThrowableBase }
type ConcurrentModificationException struct{ ThrowableBase }

// The New* constructors mirror `new X(message)` in Java. They are referenced by
// name from generated `throw` statements.

func NewError(message string) Error {
	return Error{newThrowableBase("Error", message)}
}

func NewAssertionError(message string) AssertionError {
	return AssertionError{newThrowableBase("AssertionError", message)}
}

func NewLinkageError(message string) LinkageError {
	return LinkageError{newThrowableBase("LinkageError", message)}
}

// NewExceptionInInitializerError mirrors Java's no-argument and Throwable
// constructors through one runtime entry point. Generated ordinary `new`
// expressions pass an empty string for the no-argument form; class
// initialization passes the throwable that escaped the initializer.
func NewExceptionInInitializerError(value interface{}) ExceptionInInitializerError {
	base := newThrowableBase("ExceptionInInitializerError", "")
	emptyNoArgMarker := false
	if stringValue, ok := value.(string); ok {
		emptyNoArgMarker = stringValue == ""
	}
	if value != nil && !emptyNoArgMarker {
		base.state.cause = value
	}
	return ExceptionInInitializerError{base}
}

func NewNoClassDefFoundError(message string) NoClassDefFoundError {
	return NoClassDefFoundError{newThrowableBase("NoClassDefFoundError", message)}
}

func NewNoClassDefFoundErrorWithCause(message string, cause interface{}) NoClassDefFoundError {
	base := newThrowableBase("NoClassDefFoundError", message)
	base.state.cause = cause
	return NoClassDefFoundError{base}
}

func NewException(message string) Exception {
	return Exception{newThrowableBase("Exception", message)}
}

func NewRuntimeException(message string) RuntimeException {
	return RuntimeException{newThrowableBase("RuntimeException", message)}
}

func NewIllegalArgumentException(message string) IllegalArgumentException {
	return IllegalArgumentException{newThrowableBase("IllegalArgumentException", message)}
}

// NewIllegalArgumentExceptionWithCause mirrors the two-argument JDK
// constructor used internally when Throwable.addSuppressed rejects
// self-suppression.
func NewIllegalArgumentExceptionWithCause(message string, cause interface{}) IllegalArgumentException {
	base := newThrowableBase("IllegalArgumentException", message)
	base.state.cause = cause
	return IllegalArgumentException{base}
}

func NewIllegalStateException(message string) IllegalStateException {
	return IllegalStateException{newThrowableBase("IllegalStateException", message)}
}

func NewIllegalMonitorStateException(message string) IllegalMonitorStateException {
	return IllegalMonitorStateException{newThrowableBase("IllegalMonitorStateException", message)}
}

func NewNullPointerException(message string) NullPointerException {
	return NullPointerException{newThrowableBase("NullPointerException", message)}
}

func NewNegativeArraySizeException(message string) NegativeArraySizeException {
	return NegativeArraySizeException{newThrowableBase("NegativeArraySizeException", message)}
}

func NewIndexOutOfBoundsException(message string) IndexOutOfBoundsException {
	return IndexOutOfBoundsException{newThrowableBase("IndexOutOfBoundsException", message)}
}

func NewArrayIndexOutOfBoundsException(message string) ArrayIndexOutOfBoundsException {
	return ArrayIndexOutOfBoundsException{newThrowableBase("ArrayIndexOutOfBoundsException", message)}
}

func NewArrayStoreException(message string) ArrayStoreException {
	return ArrayStoreException{newThrowableBase("ArrayStoreException", message)}
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

func NewNoSuchElementException(message string) NoSuchElementException {
	return NoSuchElementException{newThrowableBase("NoSuchElementException", message)}
}

func NewConcurrentModificationException(message string) ConcurrentModificationException {
	return ConcurrentModificationException{newThrowableBase("ConcurrentModificationException", message)}
}

func NewIOException(message string) IOException {
	return IOException{newThrowableBase("IOException", message)}
}
