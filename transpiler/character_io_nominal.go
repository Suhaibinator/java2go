package transpiler

import "strings"

var characterIONominalConstants = map[string]string{
	"Reader": "JavaIOReaderType", "Writer": "JavaIOWriterType",
	"Appendable": "JavaLangAppendableType", "Closeable": "JavaIOCloseableType", "Flushable": "JavaIOFlushableType",
}

func characterIONominalConstant(javaType string, ctx Ctx) string {
	base, _ := parseJavaTypeString(javaType)
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	name := stripJavaQualifier(base)
	constant := characterIONominalConstants[name]
	if constant == "" {
		return ""
	}
	prefix := "java.io."
	if name == "Appendable" {
		prefix = "java.lang."
	}
	if strings.Contains(base, ".") && base != prefix+name {
		return ""
	}
	return constant
}
