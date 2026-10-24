package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
)

func TestEnumEvidence191InvocationOwnershipAndRefusal(t *testing.T) {
	for _, name := range []string{"exact", "nil", "empty", "foreign-switch", "wrong-selector", "wrong-block", "file-rebound", "root-rebound", "rekeyed-root"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
			selection := selections[0]
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			evidence := newEnumSwitchLoweringEvidence(selector, block, ctx)
			source := ctx.currentFile.Source
			label := nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(block)[0])[0])[0]
			expression := parseSwitchLabelExpression(label, source, ctx, evidence)
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil || !enumSwitchStorageCorresponds(expression, label, owner.EnumConstants[0].Field, owner, ctx, evidence) {
				t.Fatal("actual emission did not establish local proof")
			}
			clause := &ast.CaseClause{List: []ast.Expr{expression}, Body: []ast.Stmt{&ast.ReturnStmt{}}}
			fallback := &ast.CaseClause{Body: []ast.Stmt{&ast.ReturnStmt{}}}
			statement := &ast.SwitchStmt{Tag: ast.NewIdent("originalTag"), Body: &ast.BlockStmt{List: []ast.Stmt{clause, fallback}}}
			switch name {
			case "nil":
				evidence = nil
			case "empty":
				evidence = newEnumSwitchLoweringEvidence(selector, block, ctx)
			case "foreign-switch":
				foreign, switches := enumReview184Context(t, enumReview184CombinedJava, "Colon")
				evidence = newEnumSwitchLoweringEvidence(switches[0].ChildByFieldName("condition"), switches[0].ChildByFieldName("body"), foreign)
			case "wrong-selector":
				evidence.selector = label
			case "wrong-block":
				evidence.block = selection
			case "file-rebound":
				foreign, _ := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
				ctx.currentFile = foreign.currentFile
			case "root-rebound":
				foreign, _ := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
				*ctx.currentFile = *foreign.currentFile
			case "rekeyed-root":
				replacement := ast.NewIdent(enumSwitch184ExpressionText(t, expression))
				evidence.storage[replacement] = evidence.storage[expression]
				expression = replacement
				clause.List[0] = expression
			}
			tag, body := statement.Tag, statement.Body
			caseBody, defaultBody := clause.Body[0], fallback.Body[0]
			imports, aliases := map[string]bool{}, map[string]string{}
			for k, v := range ctx.usedImports {
				imports[k] = v
			}
			for k, v := range ctx.importAliases {
				aliases[k] = v
			}
			diagnostics := len(Diagnostics())
			got := canonicalSourceEnumNullSwitch(statement, selector, block, source, ctx, evidence)
			if name == "exact" {
				enumDispatch184AssertSafeNullSelection(t, got)
				return
			}
			if got != statement || got.Init != nil || got.Tag != tag || got.Body != body || clause.List[0] != expression || clause.Body[0] != caseBody || fallback.Body[0] != defaultBody || !reflect.DeepEqual(imports, ctx.usedImports) || !reflect.DeepEqual(aliases, ctx.importAliases) || len(Diagnostics()) != diagnostics {
				t.Fatal("refused local evidence mutated AST/imports/diagnostics")
			}
		})
	}
}
