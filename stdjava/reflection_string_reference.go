package stdjava

// Descriptor names are VM/source identifiers, distinct from arbitrary Java
// lookup text. Their canonical references are seeded with registered metadata.
func reflectionIdentifierJavaString(name string) *JavaString {
	return JavaStringLiteralUTF16(JavaStringFromHostUTF8(name).units)
}

func reflectionJavaStringException(name string, message *JavaString) Exception {
	return Exception{newExceptionConstructorBase(name, message)}
}

// Lookup compares full UTF16 keys. Java text never crosses the lossy host-output
// boundary, including failed lookups containing isolated surrogate units.
func ClassForNameJavaString(execution *Execution, name *JavaString) *Class {
	key := JavaStringSwitchKey(name)
	var id TypeID
	javaTypeRegistry.RLock()
	identifiers := make([]TypeID, 0, len(javaTypeRegistry.types))
	for candidate := range javaTypeRegistry.types {
		identifiers = append(identifiers, candidate)
	}
	javaTypeRegistry.RUnlock()
	for _, candidate := range identifiers {
		if !isPrimitiveTypeID(candidate) && JavaStringSwitchKey(reflectionIdentifierJavaString(javaClassBinaryName(candidate))) == key {
			id = candidate
			break
		}
	}
	if id == "" {
		panic(reflectionJavaStringException("ClassNotFoundException", name))
	}
	if initialize := classDescriptor(id).Initialize; initialize != nil {
		initialize(execution)
	}
	return ClassLiteral(id)
}

func (class *Class) GetFieldJavaString(name *JavaString) *Field {
	class.TypeID()
	key := JavaStringSwitchKey(name)
	for current := class; current != nil; current = current.GetSuperclass() {
		for _, field := range classDescriptor(current.TypeID()).Fields {
			if reflectionFieldPublic(field) && JavaStringSwitchKey(field.nameJavaString) == key {
				return &Field{owner: current, descriptor: field}
			}
		}
	}
	panic(reflectionJavaStringException("NoSuchFieldException", name))
}

func (class *Class) GetDeclaredFieldJavaString(name *JavaString) *Field {
	class.TypeID()
	key := JavaStringSwitchKey(name)
	for _, field := range classDescriptor(class.TypeID()).Fields {
		if JavaStringSwitchKey(field.nameJavaString) == key {
			return &Field{owner: class, descriptor: field}
		}
	}
	panic(reflectionJavaStringException("NoSuchFieldException", name))
}

func (field *Field) GetNameJavaString() *JavaString {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	return field.descriptor.nameJavaString
}

func (class *Class) GetMethodJavaString(name *JavaString, parameters ...*Class) *Method {
	class.TypeID()
	key := JavaStringSwitchKey(name)
	if len(parameters) == 0 {
		for current := class; current != nil; current = current.GetSuperclass() {
			for _, method := range classDescriptor(current.TypeID()).Methods {
				if JavaStringSwitchKey(method.nameJavaString) == key {
					return &Method{current, method}
				}
			}
		}
	}
	panic(reflectionJavaStringException("NoSuchMethodException", name))
}

func (method *Method) GetNameJavaString() *JavaString {
	if method == nil {
		panic(NewNullPointerException("method is null"))
	}
	return method.descriptor.nameJavaString
}
