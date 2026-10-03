package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// String's generated representation is a nullable reference. Resolve Java
// bindings before choosing it: a lexical String type parameter/source class or
// an explicitly imported foreign String must not borrow java.lang.String's ABI.
func isBuiltinJavaString(javaType string, ctx Ctx) bool {
	base, args := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(args) != 0 || (base != "String" && base != "java.lang.String") {
		return false
	}
	if base == "String" {
		if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
			return false
		}
		if ctx.currentFile != nil {
			if owner, found := ctx.currentFile.Imports[base]; found && owner != "java.lang" {
				return false
			}
		}
	}
	return resolveClassScopeByQualifiedName(ctx, base) == nil
}

func javaStringReferenceType(ctx Ctx) ast.Expr {
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaString", ctx)}
}

func javaStringNullExpr(ctx Ctx) ast.Expr {
	return &ast.CallExpr{Fun: javaStringReferenceType(ctx), Args: []ast.Expr{ast.NewIdent("nil")}}
}

func javaStringLiteralUnitsExpr(units []uint16, ctx Ctx) ast.Expr {
	elements := make([]ast.Expr, len(units))
	for i, unit := range units {
		elements[i] = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(uint64(unit), 10)}
	}
	return stdjavaCall(ctx, "JavaStringLiteralUTF16", &ast.CompositeLit{Type: &ast.ArrayType{Elt: ast.NewIdent("uint16")}, Elts: elements})
}

// Decode Java content directly to UTF16. A legal isolated surrogate never
// passes through a Go Unicode scalar literal or UTF8 decoder. The parser only
// canonicalizes repeated-u spelling; Unicode and ordinary escapes remain distinct.
func javaStringLiteralUnits(raw string) ([]uint16, error) {
	textBlock := strings.HasPrefix(raw, "\"\"\"")
	content := ""
	if textBlock {
		content = stripTextBlockIncidentalWhitespace(strings.ReplaceAll(raw, "\r\n", "\n"))
	} else {
		if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
			return nil, fmt.Errorf("invalid Java String literal")
		}
		content = raw[1 : len(raw)-1]
	}
	translated, err := javaStringUnicodeUnits(content)
	if err != nil {
		return nil, err
	}
	units := make([]uint16, 0, len(translated))
	for i := 0; i < len(translated); {
		unit := translated[i]
		i++
		if unit != '\\' {
			units = append(units, unit)
			continue
		}
		if i == len(translated) {
			return nil, fmt.Errorf("trailing Java literal escape")
		}
		escaped := translated[i]
		i++
		switch escaped {
		case 'b':
			units = append(units, '\b')
		case 't':
			units = append(units, '\t')
		case 'n':
			units = append(units, '\n')
		case 'f':
			units = append(units, '\f')
		case 'r':
			units = append(units, '\r')
		case 's':
			units = append(units, ' ')
		case '\\', '\'', '"':
			units = append(units, escaped)
		case '\n':
			if !textBlock {
				return nil, fmt.Errorf("line continuation outside text block")
			}
		case '\r':
			if !textBlock {
				return nil, fmt.Errorf("line continuation outside text block")
			}
			if i < len(translated) && translated[i] == '\n' {
				i++
			}
		default:
			if escaped < '0' || escaped > '7' {
				return nil, fmt.Errorf("unsupported Java literal escape %q", rune(escaped))
			}
			value := escaped - '0'
			limit := 2
			if escaped <= '3' {
				limit = 3
			}
			for digits := 1; digits < limit && i < len(translated) && translated[i] >= '0' && translated[i] <= '7'; digits++ {
				value = value*8 + (translated[i] - '0')
				i++
			}
			units = append(units, value)
		}
	}
	return units, nil
}

// Unicode translation precedes ordinary escape decoding. An escape-produced
// backslash can introduce a Java newline escape, but Unicode results are never
// recursively rescanned as further Unicode escapes. Raw backslash eligibility
// follows the same translated-tail rule as parsing.canonicalJavaUnicodeEscapes.
func javaStringUnicodeUnits(content string) ([]uint16, error) {
	units := make([]uint16, 0, len(content))
	trailingBackslashes := 0
	lastWasEscape := false
	for i := 0; i < len(content); {
		if content[i] == '\\' && (lastWasEscape || trailingBackslashes%2 == 0) {
			if unit, end, ok := javaLiteralUnicodeUnit(content, i); ok {
				units = append(units, uint16(unit))
				i = end
				if unit == '\\' {
					trailingBackslashes++
				} else {
					trailingBackslashes = 0
				}
				lastWasEscape = true
				continue
			}
		}
		scalar, size := utf8.DecodeRuneInString(content[i:])
		if scalar == utf8.RuneError && size == 1 {
			return nil, fmt.Errorf("invalid source UTF8")
		}
		units = append(units, utf16.Encode([]rune{scalar})...)
		i += size
		if scalar == '\\' {
			trailingBackslashes++
		} else {
			trailingBackslashes = 0
		}
		lastWasEscape = false
	}
	return units, nil
}

func canonicalStringLiteral(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	units, err := javaStringLiteralUnits(node.Content(source))
	if err != nil {
		reportUnsupported("String literal: "+err.Error(), node, source, ctx)
		return &ast.BadExpr{}
	}
	return javaStringLiteralUnitsExpr(units, ctx)
}

// The primitive overload is selected from Java's static type, since rune and
// int32 have the same Go representation. Reference valueOf preserves a returned
// null; text consumers normalize it separately.
func canonicalStringValueOf(javaType string, expr ast.Expr, textOperand bool, ctx Ctx) ast.Expr {
	if primitive, ok := javaPrimitiveType(javaType); ok {
		helper := ""
		switch primitive {
		case "char":
			helper = "JavaStringValueOfChar"
		case "boolean":
			helper = "JavaStringValueOfBoolean"
		case "byte", "short", "int":
			helper = "JavaStringValueOfInt"
			expr = callIdent("int32", expr)
		case "long":
			helper = "JavaStringValueOfLong"
		case "float":
			helper = "JavaStringValueOfFloat"
		case "double":
			helper = "JavaStringValueOfDouble"
		}
		if helper != "" {
			return stdjavaCall(ctx, helper, expr)
		}
	}
	helper := "JavaStringValueOfExecution"
	if textOperand {
		helper = "JavaStringTextOperandExecution"
	}
	return stdjavaCall(ctx, helper, intrinsicExecutionExpr(ctx), expr)
}

// Only top-level case labels belong to this selector; nested switches retain
// their own type and dispatch. Key conversion rejects a null selector first.
func canonicalStringSwitch(tag ast.Expr, tagNode *sitter.Node, body *ast.BlockStmt, source []byte, ctx Ctx) ast.Expr {
	javaType, known := inferExprJavaType(tagNode, ctx, source)
	if !known || !isBuiltinJavaString(javaType, ctx) {
		return tag
	}
	for _, statement := range body.List {
		if clause, ok := statement.(*ast.CaseClause); ok {
			for i, label := range clause.List {
				clause.List[i] = stdjavaCall(ctx, "JavaStringSwitchKey", label)
			}
		}
	}
	return stdjavaCall(ctx, "JavaStringSwitchKey", tag)
}

// Resolve a local constant at its actual lexical use, rather than using the
// method-wide flattened symbol list (which contains later and sibling locals).
func javaLocalConstantExpression(node *sitter.Node, source []byte, ctx Ctx, visiting map[*symbol.Definition]bool) (bool, bool) {
	if node == nil || node.Type() != "identifier" {
		return false, false
	}
	name := node.Content(source)
	inspect := func(declaration *sitter.Node) (bool, bool) {
		if declaration == nil || declaration.Type() != "local_variable_declaration" {
			return false, false
		}
		for i := 0; i < int(declaration.NamedChildCount()); i++ {
			declarator := declaration.NamedChild(i)
			if declarator.Type() != "variable_declarator" {
				continue
			}
			ident := declarator.ChildByFieldName("name")
			if ident == nil || ident.Content(source) != name || ident.StartByte() >= node.StartByte() {
				continue
			}
			final := false
			for j := 0; j < int(declaration.NamedChildCount()); j++ {
				modifiers := declaration.NamedChild(j)
				if modifiers.Type() != "modifiers" {
					continue
				}
				for k := 0; k < int(modifiers.ChildCount()); k++ {
					final = final || modifiers.Child(k).Type() == "final"
				}
			}
			javaType := declaration.ChildByFieldName("type").Content(source)
			initializer := declarator.ChildByFieldName("value")
			// An initializer cannot use its own unassigned variable as a constant.
			if initializer == nil || node.StartByte() >= initializer.StartByte() && node.EndByte() <= initializer.EndByte() {
				return false, true
			}
			if !final || !compileTimeConstantJavaType(javaType) {
				return false, true
			}
			if isJavaStringType(javaType) && !isBuiltinJavaString(javaType, ctx) {
				return false, true
			}
			return compileTimeConstantExpression(initializer, source, ctx, visiting), true
		}
		return false, false
	}
	for child, parent := node, node.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "local_variable_declaration":
			if constant, found := inspect(parent); found {
				return constant, true
			}
		case "block", "constructor_body":
			for i := int(parent.NamedChildCount()) - 1; i >= 0; i-- {
				sibling := parent.NamedChild(i)
				if sibling.StartByte() >= child.StartByte() {
					continue
				}
				if constant, found := inspect(sibling); found {
					return constant, true
				}
			}
		case "for_statement":
			if constant, found := inspect(parent.ChildByFieldName("init")); found {
				return constant, true
			}
		case "enhanced_for_statement":
			if variable := parent.ChildByFieldName("name"); variable != nil && variable.Content(source) == name {
				return false, true
			}
		case "method_declaration", "constructor_declaration", "lambda_expression":
			if ctx.localScope != nil && ctx.localScope.ParameterByName(name) != nil {
				return false, true
			}
		}
	}
	return false, false
}
