package stdjava

// JavaURL stores immutable UTF16 components for the admitted default handlers.
// Construction and formatting never connect to a host or consult DNS.
type JavaURL struct {
	protocol, authority, path, query, ref *JavaString
}

func init()                                { RegisterJavaType("java.net.URL", ObjectTypeID, SerializableTypeID) }
func (*JavaURL) JavaDynamicTypeID() TypeID { return "java.net.URL" }

func NewJavaURLStringExecution(execution *Execution, text *JavaString) (value *JavaURL) {
	// A host filesystem callback is outside URL's semantic parser. Unwrap its
	// abrupt value only at the public boundary, after nested jar constructors
	// have carried it through their parser boundaries without reclassification.
	defer func() {
		if failure := recover(); failure != nil {
			if abrupt, ok := failure.(urlProviderAbrupt); ok {
				panic(abrupt.value)
			}
			panic(failure)
		}
	}()
	return newJavaURLStringExecution(execution, text)
}

func newJavaURLStringExecution(execution *Execution, text *JavaString) (value *JavaURL) {
	// Only parser-authored Java exceptions are wrapped. Unsupported services and
	// unrelated Go panics remain visible instead of masquerading as malformed URLs.
	defer func() {
		if failure := recover(); failure != nil {
			switch failure.(type) {
			case MalformedURLException, *MalformedURLException:
				panic(failure)
			case NumberFormatException, IllegalArgumentException, NullPointerException:
				panic(urlMalformedFromCauseExecution(execution, failure))
			default:
				panic(failure)
			}
		}
	}()
	if text == nil {
		panic(NewJavaNullPointerExceptionMessage(uriRuntimeText(`Cannot invoke "String.length()" because "spec" is null`)))
	}
	requireSystemPropertyURLProfile()
	return (&javaURLParser{input: text, execution: execution}).parse()
}

func requireJavaURL(value *JavaURL) *JavaURL {
	if value == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return value
}
func (value *JavaURL) ToExternalFormJava2goExecution(_ *Execution) *JavaString {
	value = requireJavaURL(value)
	units := value.protocol.UTF16Copy()
	units = append(units, ':')
	if value.authority != nil && len(value.authority.units) != 0 {
		units = append(units, '/', '/')
		units = append(units, value.authority.units...)
	}
	if value.path != nil {
		units = append(units, value.path.units...)
	}
	if value.query != nil {
		units = append(units, '?')
		units = append(units, value.query.units...)
	}
	if value.ref != nil {
		units = append(units, '#')
		units = append(units, value.ref.units...)
	}
	return NewJavaStringUTF16(units)
}
func (value *JavaURL) StringJava2goExecution(execution *Execution) *JavaString {
	return value.ToExternalFormJava2goExecution(execution)
}
func (value *JavaURL) EqualsJava2goExecution(_ *Execution, _ any) bool {
	requireJavaURL(value)
	panic(NewUnsupportedOperationException("java.net.URL.equals is not implemented"))
}
func (value *JavaURL) HashCodeJava2goExecution(_ *Execution) int32 {
	requireJavaURL(value)
	panic(NewUnsupportedOperationException("java.net.URL.hashCode is not implemented"))
}
