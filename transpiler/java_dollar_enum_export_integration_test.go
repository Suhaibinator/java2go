package transpiler

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestJavaDollarEnumConstantsExportAcrossPackagesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>enumexport</artifactId><version>1</version></project>`,
		"src/main/java/p/E.java": `package p;
public enum E {
 $A, small$, Small$;
 public static int inside(){return $A.ordinal()+small$.ordinal()+Small$.ordinal();}
}`,
		"src/main/java/app/Main.java": `package app;
import p.E;
import static p.E.$A;
public class Main {public static void main(String[] args) throws Exception {
 System.out.println(E.$A.name()+":"+E.small$.name()+":"+E.Small$.name()+":"+
  (E.valueOf("$A")==E.$A)+":"+(E.values()[0]==$A)+":"+
  (E.class.getField("$A").get(null)==E.$A)+":"+E.inside());
}}`,
	}, "app.Main", "$A:small$:Small$:true:true:true:3\n")
}

func TestGeneratedGoIdentifierLoweringKeepsOrdinaryGoNameText(t *testing.T) {
	literal := &ast.BasicLit{Kind: token.STRING, Value: `"cost$"`}
	file := &ast.File{Name: ast.NewIdent("main"), Decls: []ast.Decl{
		&ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("value")}, Values: []ast.Expr{
				&ast.CompositeLit{Type: ast.NewIdent("UserRecord"), Elts: []ast.Expr{
					&ast.KeyValueExpr{Key: ast.NewIdent("GoName"), Value: literal},
				}},
			}},
		}},
	}}
	lowerGeneratedGoIdentifiers(file, Ctx{})
	if literal.Value != `"cost$"` {
		t.Fatalf("ordinary GoName field text was rewritten: %s", literal.Value)
	}
}
