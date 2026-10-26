package transpiler

// Complete top-level java.base/java/lang/reflect package inventory from pinned
// OpenJDK21 src.zip d887da897fc0065a2532500ee92320ce3bc633ccd6c4e637304045ce5d90de43.
// 36 package-validated Java files; 35 top-level declarations, 33 public.
// Presence is TYPE namespace evidence, not runtime/API support. Native identity
// still requires the separate existing reflection protocol declaration set.
var declaredJavaLangReflectTypes = map[string]declaredExternalType{
	"AccessFlag":                          {public: true, isInterface: false},
	"AccessibleObject":                    {public: true, isInterface: false},
	"AnnotatedArrayType":                  {public: true, isInterface: true},
	"AnnotatedElement":                    {public: true, isInterface: true},
	"AnnotatedParameterizedType":          {public: true, isInterface: true},
	"AnnotatedType":                       {public: true, isInterface: true},
	"AnnotatedTypeVariable":               {public: true, isInterface: true},
	"AnnotatedWildcardType":               {public: true, isInterface: true},
	"Array":                               {public: true, isInterface: false},
	"ClassFileFormatVersion":              {public: true, isInterface: false},
	"Constructor":                         {public: true, isInterface: false},
	"Executable":                          {public: true, isInterface: false},
	"Field":                               {public: true, isInterface: false},
	"GenericArrayType":                    {public: true, isInterface: true},
	"GenericDeclaration":                  {public: true, isInterface: true},
	"GenericSignatureFormatError":         {public: true, isInterface: false},
	"InaccessibleObjectException":         {public: true, isInterface: false},
	"InvocationHandler":                   {public: true, isInterface: true},
	"InvocationTargetException":           {public: true, isInterface: false},
	"MalformedParameterizedTypeException": {public: true, isInterface: false},
	"MalformedParametersException":        {public: true, isInterface: false},
	"Member":                              {public: true, isInterface: true},
	"Method":                              {public: true, isInterface: false},
	"Modifier":                            {public: true, isInterface: false},
	"Parameter":                           {public: true, isInterface: false},
	"ParameterizedType":                   {public: true, isInterface: true},
	"Proxy":                               {public: true, isInterface: false},
	"ProxyGenerator":                      {public: false, isInterface: false},
	"RecordComponent":                     {public: true, isInterface: false},
	"ReflectAccess":                       {public: false, isInterface: false},
	"ReflectPermission":                   {public: true, isInterface: false},
	"Type":                                {public: true, isInterface: true},
	"TypeVariable":                        {public: true, isInterface: true},
	"UndeclaredThrowableException":        {public: true, isInterface: false},
	"WildcardType":                        {public: true, isInterface: true},
}
