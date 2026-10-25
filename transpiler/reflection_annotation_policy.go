package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type reflectionAnnotationPolicy struct {
	retention string
	targets   []string
	inherited bool
}

var reflectionPolicyEnumMembers = map[string]map[string]bool{
	"java.lang.annotation.RetentionPolicy": {"SOURCE": true, "CLASS": true, "RUNTIME": true},
	"java.lang.annotation.ElementType":     {"TYPE": true, "FIELD": true, "METHOD": true, "PARAMETER": true, "CONSTRUCTOR": true, "LOCAL_VARIABLE": true, "ANNOTATION_TYPE": true, "PACKAGE": true, "TYPE_PARAMETER": true, "TYPE_USE": true, "MODULE": true, "RECORD_COMPONENT": true},
}

// Policy helpers always receive the declaration's owning file explicitly;
// no annotation spelling or enum suffix is declaration authority.
func reflectionAnnotationValueNode(annotation *sitter.Node, source []byte) *sitter.Node {
	values := nodeutil.NamedChildrenOf(annotation.ChildByFieldName("arguments"))
	if len(values) != 1 {
		return nil
	}
	value := values[0]
	if value.Type() == "element_value_pair" {
		key := value.ChildByFieldName("key")
		if key == nil {
			key = value.ChildByFieldName("name")
		}
		if key == nil || key.Content(source) != "value" {
			return nil
		}
		return value.ChildByFieldName("value")
	}
	return value
}
func reflectionPolicyEnumValue(node *sitter.Node, owner string, ctx Ctx, source []byte) (string, bool) {
	if node == nil {
		return "", false
	}
	text := strings.TrimSpace(node.Content(source))
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	if dot := strings.LastIndexByte(text, '.'); dot >= 0 {
		value := text[dot+1:]
		return value, reflectionPolicyEnumMembers[owner][value] && declaredExternalNominalOwner(text[:dot], bindings, ctx) == owner
	}
	if !reflectionPolicyEnumMembers[owner][text] {
		return "", false
	}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		for _, field := range scope.Fields {
			if field.OriginalName == text {
				return "", false
			}
		}
	}
	selected := ""
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard && imported.member != text {
			continue
		}
		importedOwner := declaredExternalNominalOwner(imported.owner, bindings, ctx)
		members, known := reflectionPolicyEnumMembers[importedOwner]
		if !known {
			return "", false
		}
		if !members[text] {
			continue
		}
		if selected != "" && selected != importedOwner {
			return "", false
		}
		selected = importedOwner
	}
	return text, selected == owner
}
func reflectionSourceAnnotationPolicy(scope *symbol.ClassScope) (reflectionAnnotationPolicy, bool) {
	policy := reflectionAnnotationPolicy{retention: "CLASS", targets: []string{"*"}}
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "annotation_type_declaration" {
		return policy, false
	}
	file := findFileScopeForClassScope(scope)
	if file == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil || !threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot()) {
		return policy, false
	}
	annotations := declarationAnnotations(scope)
	if len(annotations) == 0 {
		return policy, true
	}
	ctx := classScopeCtx(scope, Ctx{currentFile: file, currentClass: scope})
	bindings := symbol.VisibleTypeParamBindings(scope.TypeParameters)
	for _, annotation := range annotations {
		name := annotation.ChildByFieldName("name")
		if name == nil {
			return policy, false
		}
		written := name.Content(file.Source)
		owner := declaredExternalNominalOwner(written, bindings, ctx)
		switch owner {
		case "java.lang.annotation.Inherited":
			policy.inherited = true
		case "java.lang.annotation.Retention":
			value, known := reflectionPolicyEnumValue(reflectionAnnotationValueNode(annotation, file.Source), "java.lang.annotation.RetentionPolicy", ctx, file.Source)
			if !known {
				return policy, false
			}
			policy.retention = value
		case "java.lang.annotation.Target":
			value := reflectionAnnotationValueNode(annotation, file.Source)
			if value == nil {
				return policy, false
			}
			nodes := []*sitter.Node{value}
			if value.Type() == "element_value_array_initializer" || value.Type() == "array_initializer" {
				nodes = nodeutil.NamedChildrenOf(value)
			}
			policy.targets = nil
			for _, node := range nodes {
				target, known := reflectionPolicyEnumValue(node, "java.lang.annotation.ElementType", ctx, file.Source)
				if !known {
					return policy, false
				}
				policy.targets = append(policy.targets, target)
			}
		default:
			if owner == "" && resolveClassScopeByQualifiedName(ctx, written) == nil {
				switch stripJavaQualifier(written) {
				case "Retention", "Target", "Inherited":
					return policy, false
				}
			}
		}
	}
	return policy, true
}
func reflectionAnnotationTargetApplies(policy reflectionAnnotationPolicy, node *sitter.Node) (bool, bool) {
	if node == nil {
		return false, false
	}
	context := ""
	switch node.Type() {
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
		context = "TYPE"
	case "annotation_type_declaration":
		context = "ANNOTATION_TYPE"
	case "field_declaration", "constant_declaration", "enum_constant":
		context = "FIELD"
	case "method_declaration", "annotation_type_element_declaration":
		context = "METHOD"
	case "constructor_declaration", "compact_constructor_declaration":
		context = "CONSTRUCTOR"
	default:
		return false, false
	}
	for _, target := range policy.targets {
		if target == "*" || target == context || context == "ANNOTATION_TYPE" && target == "TYPE" || (context == "TYPE" || context == "ANNOTATION_TYPE") && target == "TYPE_USE" {
			return true, true
		}
	}
	return false, true
}
func reflectionUnavailableAnnotationDescriptor(id string, policy reflectionAnnotationPolicy, ctx Ctx) ast.Expr {
	callback := sourceReflectionUnavailableCallback(ctx, &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}, "annotation proxy metadata is unavailable")
	values := []ast.Expr{metadataKey("Type", javaTypeIDLiteral(id, ctx)), metadataKey("Factory", callback)}
	if policy.inherited {
		values = append(values, metadataKey("Inherited", ast.NewIdent("true")))
	}
	return &ast.CompositeLit{Elts: values}
}
