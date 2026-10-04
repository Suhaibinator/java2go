package stdjava

// builtinThrowableDescriptors is deliberately separate from exceptionHierarchy:
// Java reference identity is qualified, while catch dispatch retains its existing
// source-name protocol. Never canonicalize arbitrary source-class descriptors.
var builtinThrowableDescriptors = map[string]struct {
	id     TypeID
	parent string
}{
	"Throwable":                       {id: "java.lang.Throwable", parent: ""},
	"Error":                           {id: "java.lang.Error", parent: "Throwable"},
	"AssertionError":                  {id: "java.lang.AssertionError", parent: "Error"},
	"LinkageError":                    {id: "java.lang.LinkageError", parent: "Error"},
	"ExceptionInInitializerError":     {id: "java.lang.ExceptionInInitializerError", parent: "LinkageError"},
	"NoClassDefFoundError":            {id: "java.lang.NoClassDefFoundError", parent: "LinkageError"},
	"CloneNotSupportedException":      {id: "java.lang.CloneNotSupportedException", parent: "Exception"},
	"Exception":                       {id: "java.lang.Exception", parent: "Throwable"},
	"RuntimeException":                {id: "java.lang.RuntimeException", parent: "Exception"},
	"IllegalArgumentException":        {id: "java.lang.IllegalArgumentException", parent: "RuntimeException"},
	"IllegalStateException":           {id: "java.lang.IllegalStateException", parent: "RuntimeException"},
	"IllegalMonitorStateException":    {id: "java.lang.IllegalMonitorStateException", parent: "RuntimeException"},
	"IllegalThreadStateException":     {id: "java.lang.IllegalThreadStateException", parent: "IllegalArgumentException"},
	"NullPointerException":            {id: "java.lang.NullPointerException", parent: "RuntimeException"},
	"NegativeArraySizeException":      {id: "java.lang.NegativeArraySizeException", parent: "RuntimeException"},
	"IndexOutOfBoundsException":       {id: "java.lang.IndexOutOfBoundsException", parent: "RuntimeException"},
	"ArrayIndexOutOfBoundsException":  {id: "java.lang.ArrayIndexOutOfBoundsException", parent: "IndexOutOfBoundsException"},
	"StringIndexOutOfBoundsException": {id: "java.lang.StringIndexOutOfBoundsException", parent: "IndexOutOfBoundsException"},
	"ArrayStoreException":             {id: "java.lang.ArrayStoreException", parent: "RuntimeException"},
	"NumberFormatException":           {id: "java.lang.NumberFormatException", parent: "IllegalArgumentException"},
	"ArithmeticException":             {id: "java.lang.ArithmeticException", parent: "RuntimeException"},
	"ClassCastException":              {id: "java.lang.ClassCastException", parent: "RuntimeException"},
	"UnsupportedOperationException":   {id: "java.lang.UnsupportedOperationException", parent: "RuntimeException"},
	"InterruptedException":            {id: "java.lang.InterruptedException", parent: "Exception"},
	"ReflectiveOperationException":    {id: "java.lang.ReflectiveOperationException", parent: "Exception"},
	"ClassNotFoundException":          {id: "java.lang.ClassNotFoundException", parent: "ReflectiveOperationException"},
	"NoSuchMethodException":           {id: "java.lang.NoSuchMethodException", parent: "ReflectiveOperationException"},
	"NoSuchFieldException":            {id: "java.lang.NoSuchFieldException", parent: "ReflectiveOperationException"},
	"IllegalAccessException":          {id: "java.lang.IllegalAccessException", parent: "ReflectiveOperationException"},
	"IOException":                     {id: "java.io.IOException", parent: "Exception"},
	"FileNotFoundException":           {id: "java.io.FileNotFoundException", parent: "IOException"},
	"UnsupportedEncodingException":    {id: "java.io.UnsupportedEncodingException", parent: "IOException"},
	"NoSuchElementException":          {id: "java.util.NoSuchElementException", parent: "RuntimeException"},
	"ConcurrentModificationException": {id: "java.util.ConcurrentModificationException", parent: "RuntimeException"},
	"ExecutionException":              {id: "java.util.concurrent.ExecutionException", parent: "Exception"},
	"TimeoutException":                {id: "java.util.concurrent.TimeoutException", parent: "Exception"},
	"CancellationException":           {id: "java.util.concurrent.CancellationException", parent: "IllegalStateException"},
	"RejectedExecutionException":      {id: "java.util.concurrent.RejectedExecutionException", parent: "RuntimeException"},
	"ParseException":                  {id: "java.text.ParseException", parent: "Exception"},
	"NoSuchAlgorithmException":        {id: "java.security.NoSuchAlgorithmException", parent: "Exception"},
	"BufferOverflowException":         {id: "java.nio.BufferOverflowException", parent: "RuntimeException"},
	"BufferUnderflowException":        {id: "java.nio.BufferUnderflowException", parent: "RuntimeException"},
	"ReadOnlyBufferException":         {id: "java.nio.ReadOnlyBufferException", parent: "UnsupportedOperationException"},
	"AsynchronousCloseException":      {id: "java.nio.channels.AsynchronousCloseException", parent: "ClosedChannelException"},
	"ClosedByInterruptException":      {id: "java.nio.channels.ClosedByInterruptException", parent: "AsynchronousCloseException"},
	"ClosedChannelException":          {id: "java.nio.channels.ClosedChannelException", parent: "IOException"},
	"NonWritableChannelException":     {id: "java.nio.channels.NonWritableChannelException", parent: "IllegalStateException"},
	"IllegalCharsetNameException":     {id: "java.nio.charset.IllegalCharsetNameException", parent: "IllegalArgumentException"},
	"UnsupportedCharsetException":     {id: "java.nio.charset.UnsupportedCharsetException", parent: "IllegalArgumentException"},
	"CharacterCodingException":        {id: "java.nio.charset.CharacterCodingException", parent: "IOException"},
	"MalformedInputException":         {id: "java.nio.charset.MalformedInputException", parent: "CharacterCodingException"},
	"UnmappableCharacterException":    {id: "java.nio.charset.UnmappableCharacterException", parent: "CharacterCodingException"},
	"InvocationTargetException":       {id: "java.lang.reflect.InvocationTargetException", parent: "ReflectiveOperationException"},
}

// BuiltinThrowableTypeID resolves a known runtime-owned exception name. The
// compiler calls this only after ruling out a source declaration with that name.
// Qualified names and unknown names are retained exactly.
func BuiltinThrowableTypeID(name string) TypeID {
	if descriptor, ok := builtinThrowableDescriptors[name]; ok {
		return descriptor.id
	}
	return TypeID(name)
}

func init() {
	for _, descriptor := range builtinThrowableDescriptors {
		parent := ObjectTypeID
		if descriptor.parent != "" {
			parent = BuiltinThrowableTypeID(descriptor.parent)
		}
		if descriptor.id == ThrowableTypeID {
			RegisterJavaType(descriptor.id, parent, SerializableTypeID)
		} else {
			RegisterJavaType(descriptor.id, parent)
		}
	}
}
