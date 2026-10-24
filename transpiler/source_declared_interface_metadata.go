package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// sourceDeclaredExternalInterfaceIDs records direct nominal edges only. Source
// interfaces remain owned by the existing source resolver; their registrations
// supply transitive edges. A generated Go method shape is never identity proof.
func sourceDeclaredExternalInterfaceIDs(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	var file *symbol.FileScope
	for owner := scope; owner != nil; owner = owner.Enclosing {
		if owned := findFileScopeForClassScope(owner, ctx); owned != nil {
			file = owned
			break
		}
	}
	if file == nil {
		return nil
	} // A caller file is not declaration-origin proof.
	declaring := classHeaderTypeCtx(scope, ctx)
	declaring.currentFile = file
	// Caller synthetic binders do not occur in the declaration's Java header.
	// Captured lexical binders are already retained on the actual class scope.
	declaring.syntheticTypeParameters = nil
	if declaring.currentFile == nil {
		return nil
	}
	written := append([]string(nil), scope.ImplementedInterfaces...)
	if node := scope.Class.DeclarationNode; node.Type() == "object_creation_expression" {
		// Anonymous platform interfaces need not appear in the source-interface
		// inventory. Their actual allocation TYPE node still owns the declaration.
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			written = append(written, typeNode.Content(declaring.currentFile.Source))
		}
	}
	seen := map[string]bool{}
	var result []ast.Expr
	for _, edge := range written {
		if owner := declarationOwnedExternalInterface(edge, scope, declaring); owner != "" && !seen[owner] {
			seen[owner] = true
			result = append(result, javaTypeIDLiteral(owner, ctx))
		}
	}
	return result
}

func declarationOwnedExternalInterface(written string, scope *symbol.ClassScope, ctx Ctx) string {
	base, _, valid := reflectiveNamedTypeParts(strings.TrimSpace(written))
	if !valid || strings.ContainsAny(base, "[]? ") || base == "" {
		return ""
	}
	first := strings.SplitN(base, ".", 2)[0]
	for owner := scope; owner != nil; owner = owner.Enclosing {
		for _, parameter := range owner.TypeParameters {
			sourceName := parameter.Name
			if parameter.Declaration != nil {
				sourceName = parameter.Declaration.SourceName
			}
			if first == sourceName {
				return ""
			}
		}
	}
	// Full source lookup covers classes as well as interfaces. A source type
	// prefix continues to own a failed member path; it cannot become a package.
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	if strings.Contains(base, ".") {
		if resolveClassScopeByQualifiedName(ctx, first) != nil {
			return ""
		}
		if _, imported := ctx.currentFile.Imports[first]; imported {
			return ""
		}
		if !declaredInterfaceDemandPrefixAbsent(first, ctx) {
			return ""
		}
		if knownDeclaredExternalInterface(base) {
			return base
		}
		return "" // Unknown nested external spellings have no binary-name proof.
	}
	if imported, present := ctx.currentFile.Imports[base]; present {
		canonical := imported + "." + base
		if knownDeclaredExternalInterface(canonical) {
			return canonical
		}
		return ""
	}
	candidate := ""
	if knownDeclaredExternalInterface("java.lang." + base) {
		candidate = "java.lang." + base
	}
	// Demand imports share an ambiguity tier with implicit java.lang. A missing
	// catalog entry proves absence only for the complete pinned package below;
	// an unknown external namespace cannot establish that absence.
	for _, owner := range ordinaryTypeOnDemandImports(ctx) {
		if owner == "java.util" {
			declaration, present := declaredJavaUtilTypes[base]
			if !present || !declaration.public {
				continue
			}
			canonical := owner + "." + base
			if !declaration.isInterface || candidate != "" && candidate != canonical {
				return ""
			}
			candidate = canonical
			continue
		}
		if owner == "java.lang" && candidate != "" {
			continue
		}
		if sourceOwner := findQualifiedSourceClass(owner); sourceOwner != nil {
			continue
		}
		if symbol.GlobalScope.FindPackage(owner) != nil {
			continue
		}
		return ""
	}
	for _, imported := range staticMethodImports(ctx) {
		if imported.wildcard && findQualifiedSourceClass(imported.owner) == nil {
			return ""
		}
	}
	return candidate
}

// A qualified spelling still begins in Java's TYPE namespace. In particular,
// an unknown demand import might expose a class named java. Package spelling
// alone cannot prove that the prefix is unbound.
func declaredInterfaceDemandPrefixAbsent(first string, ctx Ctx) bool {
	for _, owner := range ordinaryTypeOnDemandImports(ctx) {
		if owner == "java.lang.reflect" {
			if declaration, present := declaredJavaLangReflectTypes[first]; present && declaration.public {
				return false
			}
			continue
		}
		if owner == "java.util" {
			if declaration, present := declaredJavaUtilTypes[first]; present && declaration.public {
				return false
			}
			continue
		}
		if _, presence := lookupJDK21TopLevel(owner, first); presence != jdk21NominalUnknown {
			if presence != jdk21NominalAbsentCertified {
				return false
			}
			continue
		}
		if findQualifiedSourceClass(owner) != nil || symbol.GlobalScope.FindPackage(owner) != nil {
			continue
		}
		return false
	}
	for _, imported := range staticMethodImports(ctx) {
		if imported.wildcard && findQualifiedSourceClass(imported.owner) == nil {
			return false
		}
	}
	return true
}

func knownDeclaredExternalInterface(owner string) bool {
	switch owner {
	case "java.lang.Comparable", "java.lang.CharSequence", "java.lang.Cloneable", "java.lang.Iterable", "java.lang.AutoCloseable", "java.lang.Appendable", "java.lang.Readable", "java.lang.Runnable", "java.io.Serializable":
		return true
	}
	if strings.HasPrefix(owner, "java.util.") {
		declaration, known := declaredJavaUtilTypes[strings.TrimPrefix(owner, "java.util.")]
		return known && declaration.public && declaration.isInterface
	}
	return false
}

type declaredExternalType struct{ public, isInterface bool }

// Generated bootstrap inventory: OpenJDK 21.0.6+8 java.base/java/util/*.java.
// src.zip SHA256: d887da897fc0065a2532500ee92320ce3bc633ccd6c4e637304045ce5d90de43
// Inventory SHA256: 9bfb6a5a62be1f5604a658e916575746d595e15bd8b98d5526594cd9be9058b1
// All 135 package-validated source members, 138 top-level declarations (110
// public). Nested declarations are excluded; duplicate top-level names rejected.
// No installed JDK files are read during transpilation or generated execution.
var declaredJavaUtilTypes = map[string]declaredExternalType{
	"AbstractCollection":                     {public: true, isInterface: false},
	"AbstractList":                           {public: true, isInterface: false},
	"AbstractMap":                            {public: true, isInterface: false},
	"AbstractQueue":                          {public: true, isInterface: false},
	"AbstractSequentialList":                 {public: true, isInterface: false},
	"AbstractSet":                            {public: true, isInterface: false},
	"ArrayDeque":                             {public: true, isInterface: false},
	"ArrayList":                              {public: true, isInterface: false},
	"ArrayPrefixHelpers":                     {public: false, isInterface: false},
	"Arrays":                                 {public: true, isInterface: false},
	"ArraysParallelSortHelpers":              {public: false, isInterface: false},
	"Base64":                                 {public: true, isInterface: false},
	"BitSet":                                 {public: true, isInterface: false},
	"Calendar":                               {public: true, isInterface: false},
	"CollSer":                                {public: false, isInterface: false},
	"Collection":                             {public: true, isInterface: true},
	"Collections":                            {public: true, isInterface: false},
	"ComparableTimSort":                      {public: false, isInterface: false},
	"Comparator":                             {public: true, isInterface: true},
	"Comparators":                            {public: false, isInterface: false},
	"ConcurrentModificationException":        {public: true, isInterface: false},
	"Currency":                               {public: true, isInterface: false},
	"Date":                                   {public: true, isInterface: false},
	"DecimalDigits":                          {public: false, isInterface: false},
	"Deque":                                  {public: true, isInterface: true},
	"Dictionary":                             {public: true, isInterface: false},
	"Digits":                                 {public: false, isInterface: true},
	"DoubleSummaryStatistics":                {public: true, isInterface: false},
	"DualPivotQuicksort":                     {public: false, isInterface: false},
	"DuplicateFormatFlagsException":          {public: true, isInterface: false},
	"EmptyStackException":                    {public: true, isInterface: false},
	"EnumMap":                                {public: true, isInterface: false},
	"EnumSet":                                {public: true, isInterface: false},
	"Enumeration":                            {public: true, isInterface: true},
	"EventListener":                          {public: true, isInterface: true},
	"EventListenerProxy":                     {public: true, isInterface: false},
	"EventObject":                            {public: true, isInterface: false},
	"FormatFlagsConversionMismatchException": {public: true, isInterface: false},
	"FormatItem":                             {public: false, isInterface: false},
	"FormatProcessor":                        {public: true, isInterface: false},
	"Formattable":                            {public: true, isInterface: true},
	"FormattableFlags":                       {public: true, isInterface: false},
	"Formatter":                              {public: true, isInterface: false},
	"FormatterBuilder":                       {public: false, isInterface: false},
	"FormatterClosedException":               {public: true, isInterface: false},
	"GregorianCalendar":                      {public: true, isInterface: false},
	"HashMap":                                {public: true, isInterface: false},
	"HashSet":                                {public: true, isInterface: false},
	"Hashtable":                              {public: true, isInterface: false},
	"HexDigits":                              {public: false, isInterface: false},
	"HexFormat":                              {public: true, isInterface: false},
	"IdentityHashMap":                        {public: true, isInterface: false},
	"IllegalFormatArgumentIndexException":    {public: false, isInterface: false},
	"IllegalFormatCodePointException":        {public: true, isInterface: false},
	"IllegalFormatConversionException":       {public: true, isInterface: false},
	"IllegalFormatException":                 {public: true, isInterface: false},
	"IllegalFormatFlagsException":            {public: true, isInterface: false},
	"IllegalFormatPrecisionException":        {public: true, isInterface: false},
	"IllegalFormatWidthException":            {public: true, isInterface: false},
	"IllformedLocaleException":               {public: true, isInterface: false},
	"ImmutableCollections":                   {public: false, isInterface: false},
	"InputMismatchException":                 {public: true, isInterface: false},
	"IntSummaryStatistics":                   {public: true, isInterface: false},
	"InvalidPropertiesFormatException":       {public: true, isInterface: false},
	"Iterator":                               {public: true, isInterface: true},
	"JapaneseImperialCalendar":               {public: false, isInterface: false},
	"JumboEnumSet":                           {public: false, isInterface: false},
	"KeyValueHolder":                         {public: false, isInterface: false},
	"LinkedHashMap":                          {public: true, isInterface: false},
	"LinkedHashSet":                          {public: true, isInterface: false},
	"LinkedList":                             {public: true, isInterface: false},
	"List":                                   {public: true, isInterface: true},
	"ListIterator":                           {public: true, isInterface: true},
	"ListResourceBundle":                     {public: true, isInterface: false},
	"Locale":                                 {public: true, isInterface: false},
	"LocaleISOData":                          {public: false, isInterface: false},
	"LongSummaryStatistics":                  {public: true, isInterface: false},
	"Map":                                    {public: true, isInterface: true},
	"MissingFormatArgumentException":         {public: true, isInterface: false},
	"MissingFormatWidthException":            {public: true, isInterface: false},
	"MissingResourceException":               {public: true, isInterface: false},
	"NavigableMap":                           {public: true, isInterface: true},
	"NavigableSet":                           {public: true, isInterface: true},
	"NoSuchElementException":                 {public: true, isInterface: false},
	"Objects":                                {public: true, isInterface: false},
	"Observable":                             {public: true, isInterface: false},
	"Observer":                               {public: true, isInterface: true},
	"OctalDigits":                            {public: false, isInterface: false},
	"Optional":                               {public: true, isInterface: false},
	"OptionalDouble":                         {public: true, isInterface: false},
	"OptionalInt":                            {public: true, isInterface: false},
	"OptionalLong":                           {public: true, isInterface: false},
	"PrimitiveIterator":                      {public: true, isInterface: true},
	"PriorityQueue":                          {public: true, isInterface: false},
	"Properties":                             {public: true, isInterface: false},
	"PropertyPermission":                     {public: true, isInterface: false},
	"PropertyPermissionCollection":           {public: false, isInterface: false},
	"PropertyResourceBundle":                 {public: true, isInterface: false},
	"Queue":                                  {public: true, isInterface: true},
	"Random":                                 {public: true, isInterface: false},
	"RandomAccess":                           {public: true, isInterface: true},
	"RegularEnumSet":                         {public: false, isInterface: false},
	"ResourceBundle":                         {public: true, isInterface: false},
	"ReverseOrderDequeView":                  {public: false, isInterface: false},
	"ReverseOrderListView":                   {public: false, isInterface: false},
	"ReverseOrderSortedMapView":              {public: false, isInterface: false},
	"ReverseOrderSortedSetView":              {public: false, isInterface: false},
	"Scanner":                                {public: true, isInterface: false},
	"SequencedCollection":                    {public: true, isInterface: true},
	"SequencedMap":                           {public: true, isInterface: true},
	"SequencedSet":                           {public: true, isInterface: true},
	"ServiceConfigurationError":              {public: true, isInterface: false},
	"ServiceLoader":                          {public: true, isInterface: false},
	"Set":                                    {public: true, isInterface: true},
	"SimpleTimeZone":                         {public: true, isInterface: false},
	"SortedMap":                              {public: true, isInterface: true},
	"SortedSet":                              {public: true, isInterface: true},
	"Spliterator":                            {public: true, isInterface: true},
	"Spliterators":                           {public: true, isInterface: false},
	"SplittableRandom":                       {public: true, isInterface: false},
	"Stack":                                  {public: true, isInterface: false},
	"StringJoiner":                           {public: true, isInterface: false},
	"StringTokenizer":                        {public: true, isInterface: false},
	"TaskQueue":                              {public: false, isInterface: false},
	"TimSort":                                {public: false, isInterface: false},
	"TimeZone":                               {public: true, isInterface: false},
	"Timer":                                  {public: true, isInterface: false},
	"TimerTask":                              {public: true, isInterface: false},
	"TimerThread":                            {public: false, isInterface: false},
	"TooManyListenersException":              {public: true, isInterface: false},
	"TreeMap":                                {public: true, isInterface: false},
	"TreeSet":                                {public: true, isInterface: false},
	"Tripwire":                               {public: false, isInterface: false},
	"UUID":                                   {public: true, isInterface: false},
	"UnknownFormatConversionException":       {public: true, isInterface: false},
	"UnknownFormatFlagsException":            {public: true, isInterface: false},
	"Vector":                                 {public: true, isInterface: false},
	"WeakHashMap":                            {public: true, isInterface: false},
}
