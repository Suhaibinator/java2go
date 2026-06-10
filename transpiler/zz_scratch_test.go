package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"testing"
)

func TestScratchConcurrency(t *testing.T) {
	src := `
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.ConcurrentHashMap;
public class Conc {
    private AtomicInteger counter = new AtomicInteger(0);
    private final Object lock = new Object();
    private int total = 0;
    public void bump() {
        counter.incrementAndGet();
        synchronized (lock) {
            total = total + 1;
        }
    }
    public int get() { return counter.get(); }
    public static void pause() throws InterruptedException {
        Thread.sleep(10);
    }
    public static void runThread() {
        Thread t = new Thread(() -> System.out.println("hi"));
        t.start();
        t.join();
    }
    public static void mapUse() {
        ConcurrentHashMap<String, Integer> m = new ConcurrentHashMap<>();
        m.put("a", 1);
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), node.(*ast.File))
	t.Log("\n" + buf.String())
}
