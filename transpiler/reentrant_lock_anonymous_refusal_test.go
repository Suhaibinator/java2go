package transpiler

import "testing"

func TestReentrantLock180AnonymousConstructorRefusal(t *testing.T) {
	const source = `import java.util.concurrent.locks.ReentrantLock;
class Use { Object create(){return new ReentrantLock(){public void lock(){throw new IllegalStateException("source-body");}};} }`
	helper := setupParseHelper(t, source)
	node := findNode(helper.File.Ast, "object_creation_expression")
	t.Logf("actual class_body=%t named-body-field=%t", objectCreationClassBody(node) != nil, node.ChildByFieldName("body") != nil)
	if reentrantLockConstructorApplicable(node, nil, nil, helper.Ctx, []byte(source)) {
		t.Fatal("held anonymous source subclass must not be admitted as canonical noarg allocation")
	}
}
