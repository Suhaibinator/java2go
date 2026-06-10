package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// intLit builds an integer literal expression, used to supply the default value
// for a no-arg atomic constructor (Java defaults to 0).
func intLit(n int) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(n)}
}

// concurrencyRuntimeTypes lists the stdjava-backed concurrency types and whether
// each is generic (so its declared type takes type arguments).
var concurrencyRuntimeTypes = map[string]bool{
	"AtomicInteger":     false,
	"AtomicLong":        false,
	"AtomicBoolean":     false,
	"Thread":            false,
	"ExecutorService":   false,
	"ConcurrentHashMap": true,
}

// stdjavaRuntimeTypeExpr maps a Java concurrency type name to its stdjava
// runtime Go type (e.g. AtomicInteger -> *stdjava.AtomicInteger,
// ConcurrentHashMap<K,V> -> *stdjava.ConcurrentHashMap[K, V]). It returns
// (nil, false) for any other name, and never fires for a user-defined class of
// the same name. Generic args are themselves lowered through
// javaTypeStringToGoTypeExpr.
func stdjavaRuntimeTypeExpr(baseName string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	// java.lang.Object maps to the empty interface. It commonly appears as the
	// type of a lock token; new Object() is handled by its constructor intrinsic.
	if baseName == "Object" && resolveClassScopeByQualifiedName(ctx, baseName) == nil {
		return &ast.Ident{Name: "any"}, true
	}

	generic, ok := concurrencyRuntimeTypes[baseName]
	if !ok {
		return nil, false
	}
	if resolveClassScopeByQualifiedName(ctx, baseName) != nil {
		return nil, false
	}

	base := stdjavaQualifiedExpr(baseName, ctx)
	if generic && len(typeArgs) > 0 {
		argExprs := make([]ast.Expr, 0, len(typeArgs))
		for _, arg := range typeArgs {
			argExprs = append(argExprs, javaTypeStringToGoTypeExpr(arg, typeParams, ctx))
		}
		base = applyTypeArguments(base, argExprs)
	}
	return &ast.StarExpr{X: base}, true
}

// This file registers java.util.concurrent and java.lang.Thread intrinsics onto
// the shared intrinsics tables (machinery in intrinsics.go). They rewrite calls
// onto the stdjava concurrency runtime (stdjava/concurrent.go):
//
//   new AtomicInteger(0)            -> stdjava.NewAtomicInteger(0)
//   counter.incrementAndGet()      -> counter.IncrementAndGet()
//   Thread.sleep(100)              -> stdjava.ThreadSleep(100)
//   new Thread(r).start()          -> stdjava.NewThread(r).Start()
//   Executors.newFixedThreadPool(4)-> stdjava.NewFixedThreadPool(4)
//
// Atomic/Thread/ExecutorService/ConcurrentHashMap instances are stdjava pointer
// types whose methods are named with Go's exported casing, so the instance
// intrinsics simply re-case the method name onto the receiver.

func init() {
	registerAtomicIntrinsics()
	registerThreadIntrinsics()
	registerExecutorIntrinsics()
	registerConcurrentMapIntrinsics()
	registerObjectMonitorIntrinsics()
}

func registerObjectMonitorIntrinsics() {
	// new Object() -> stdjava.NewObject() (a unique lock token).
	registerConstructorIntrinsic("Object", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "NewObject")
	})

	// wait()/notify()/notifyAll() on a lock typed as Object route to the stdjava
	// monitor's condition variable. These fire when the receiver's Java type
	// resolves to Object (the common `Object lock` idiom). wait(millis) is not
	// modelled; only the zero-arg forms match.
	monitorMethods := map[string]string{
		"wait":      "MonitorWait",
		"notify":    "MonitorNotify",
		"notifyAll": "MonitorNotifyAll",
	}
	for javaMethod, runtimeFn := range monitorMethods {
		runtimeFn := runtimeFn
		registerInstanceIntrinsic("Object", javaMethod, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if recv == nil || len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, runtimeFn, recv)
		})
	}
}

// extendsBuiltinThread reports whether the class scope directly extends
// java.lang.Thread (and not a user-defined class of that name).
func extendsBuiltinThread(ctx Ctx) bool {
	return classScopeExtendsThread(ctx, ctx.currentClass)
}

// classScopeExtendsThread reports whether scope transitively extends
// java.lang.Thread.
func classScopeExtendsThread(ctx Ctx, scope *symbol.ClassScope) bool {
	for scope != nil {
		super, _ := parseJavaTypeString(strings.TrimSpace(scope.Superclass))
		base := stripJavaQualifier(super)
		if base == "" {
			return false
		}
		parent := resolveClassScopeByQualifiedName(ctx, super)
		if base == "Thread" && parent == nil {
			return true
		}
		scope = parent
	}
	return false
}

// threadSubclassMethod maps a zero-arg Thread method called on a receiver whose
// Java type extends Thread onto the exported method promoted from the embedded
// *stdjava.Thread (start->Start, join->Join, run->Run). It returns ("", false)
// when the receiver is not a Thread subclass or the method is not one of these.
func threadSubclassMethod(objectNode *sitter.Node, methodName string, ctx Ctx, source []byte) (string, bool) {
	goMethod, ok := map[string]string{"start": "Start", "join": "Join", "run": "Run"}[methodName]
	if !ok {
		return "", false
	}
	javaType, ok := inferExprJavaType(objectNode, ctx, source)
	if !ok {
		return "", false
	}
	base, _ := parseJavaTypeString(javaType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if scope == nil || !classScopeExtendsThread(ctx, scope) {
		return "", false
	}
	return goMethod, true
}

// threadBaseWiringStmt builds `self.Thread = stdjava.NewThreadBase(self)` for a
// Thread subclass constructor, so the embedded runtime Thread dispatches Start()
// to this instance's Run() override. Returns nil for non-Thread classes.
func threadBaseWiringStmt(ctx Ctx) ast.Stmt {
	if !extendsBuiltinThread(ctx) {
		return nil
	}
	recv := ShortName(ctx.className)
	return &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: recv}, Sel: &ast.Ident{Name: "Thread"}}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{stdjavaCall(ctx, "NewThreadBase", &ast.Ident{Name: recv})},
	}
}

// synchronizedMethodPrologue builds the statements that acquire and defer the
// release of a synchronized method's monitor. An instance method locks its
// receiver; a static method locks a class-level token keyed by the generated
// type name. The acquired mutex is stored in a uniquely-named local so the
// deferred MonitorExit can release it.
func synchronizedMethodPrologue(ctx Ctx, static bool) []ast.Stmt {
	monName := "__java2goMethodMonitor"
	var enter ast.Expr
	if static {
		enter = stdjavaCall(ctx, "ClassMonitorEnter",
			&ast.BasicLit{Kind: token.STRING, Value: `"` + ctx.className + `"`})
	} else {
		enter = stdjavaCall(ctx, "MonitorEnter", &ast.Ident{Name: ShortName(ctx.className)})
	}
	return []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: monName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{enter},
		},
		&ast.DeferStmt{
			Call: stdjavaCall(ctx, "MonitorExit", &ast.Ident{Name: monName}),
		},
	}
}

// selectorCall builds recv.MethodName(args...), used to map a Java instance
// method onto its stdjava (exported-cased) counterpart.
func selectorCall(recv ast.Expr, method string, args []ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: method}},
		Args: args,
	}
}

// registerAtomicMethods wires the instance methods shared by the atomic wrapper
// types onto their exported stdjava equivalents.
func registerAtomicMethods(javaType string) {
	methods := map[string]string{
		"get":             "Get",
		"set":             "Set",
		"incrementAndGet": "IncrementAndGet",
		"decrementAndGet": "DecrementAndGet",
		"getAndIncrement": "GetAndIncrement",
		"getAndDecrement": "GetAndDecrement",
		"addAndGet":       "AddAndGet",
		"getAndAdd":       "GetAndAdd",
		"compareAndSet":   "CompareAndSet",
	}
	for javaMethod, goMethod := range methods {
		goMethod := goMethod
		registerInstanceIntrinsic(javaType, javaMethod, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if recv == nil {
				return nil
			}
			return selectorCall(recv, goMethod, args)
		})
	}
}

func registerAtomicIntrinsics() {
	// Constructors: new AtomicInteger(n) -> stdjava.NewAtomicInteger(n). A no-arg
	// Java constructor defaults to 0/false, so supply the zero value.
	registerConstructorIntrinsic("AtomicInteger", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return stdjavaCall(ctx, "NewAtomicInteger", intLit(0))
		}
		return stdjavaCall(ctx, "NewAtomicInteger", args[0])
	})
	registerConstructorIntrinsic("AtomicLong", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return stdjavaCall(ctx, "NewAtomicLong", intLit(0))
		}
		return stdjavaCall(ctx, "NewAtomicLong", args[0])
	})
	registerConstructorIntrinsic("AtomicBoolean", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return stdjavaCall(ctx, "NewAtomicBoolean", &ast.Ident{Name: "false"})
		}
		return stdjavaCall(ctx, "NewAtomicBoolean", args[0])
	})

	registerAtomicMethods("AtomicInteger")
	registerAtomicMethods("AtomicLong")
	// AtomicBoolean only supports get/set/compareAndSet, but reusing the table is
	// harmless: the extra method names never appear on an AtomicBoolean receiver.
	registerAtomicMethods("AtomicBoolean")
}

func registerThreadIntrinsics() {
	// new Thread(runnable) -> stdjava.NewThread(runnable). The Runnable argument
	// is already a func() in generated code (lambda or method reference).
	registerConstructorIntrinsic("Thread", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewThread", args[0])
	})

	// Thread.sleep(ms) -> stdjava.ThreadSleep(ms)
	registerStaticIntrinsic("Thread", "sleep", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "ThreadSleep", args[0])
	})

	for javaMethod, goMethod := range map[string]string{"start": "Start", "join": "Join"} {
		goMethod := goMethod
		registerInstanceIntrinsic("Thread", javaMethod, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if recv == nil || len(args) != 0 {
				return nil
			}
			return selectorCall(recv, goMethod, nil)
		})
	}
}

func registerExecutorIntrinsics() {
	// Executors.newFixedThreadPool(n) -> stdjava.NewFixedThreadPool(n)
	registerStaticIntrinsic("Executors", "newFixedThreadPool", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewFixedThreadPool", args[0])
	})
	// A single-thread executor is just a pool of size 1.
	registerStaticIntrinsic("Executors", "newSingleThreadExecutor", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "NewFixedThreadPool", intLit(1))
	})

	for javaMethod, goMethod := range map[string]string{
		"submit":           "Submit",
		"execute":          "Submit",
		"shutdown":         "Shutdown",
		"awaitTermination": "AwaitTermination",
	} {
		goMethod := goMethod
		registerInstanceIntrinsic("ExecutorService", javaMethod, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if recv == nil {
				return nil
			}
			// awaitTermination(timeout, unit) in Java takes args; the stdjava shim
			// waits unconditionally, so drop them.
			if goMethod == "AwaitTermination" {
				return selectorCall(recv, goMethod, nil)
			}
			if goMethod == "Submit" {
				if len(args) != 1 {
					return nil
				}
				return selectorCall(recv, goMethod, args)
			}
			return selectorCall(recv, goMethod, nil)
		})
	}
}

func registerConcurrentMapIntrinsics() {
	registerConstructorIntrinsic("ConcurrentHashMap", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		// Go cannot infer the type parameters of a no-arg generic constructor, so
		// pass them explicitly: stdjava.NewConcurrentHashMap[K, V]().
		return stdjavaGenericCall(ctx, "NewConcurrentHashMap", typeArgs, nil)
	})

	for javaMethod, goMethod := range map[string]string{
		"put":         "Put",
		"get":         "Get",
		"remove":      "Remove",
		"containsKey": "ContainsKey",
		"size":        "Size",
	} {
		goMethod := goMethod
		registerInstanceIntrinsic("ConcurrentHashMap", javaMethod, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if recv == nil {
				return nil
			}
			return selectorCall(recv, goMethod, args)
		})
	}
}
