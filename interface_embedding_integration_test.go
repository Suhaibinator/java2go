package main

import (
	"strings"
	"testing"
)

func TestInterfaceEmbedding_SingleParent(t *testing.T) {
	src := `
package embed;
public interface Animal { void eat(); }
public interface Pet extends Animal { void play(); }
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type Pet interface { Animal") {
		t.Fatalf("expected Pet to embed Animal interface, got:\n%s", out)
	}
	if !strings.Contains(flat, "play(") {
		t.Fatalf("expected Pet to include its own methods, got:\n%s", out)
	}
}

func TestInterfaceEmbedding_MultipleParentsWithTypeArgs(t *testing.T) {
	src := `
package embed.multi;
public interface Stream<T> { T next(); }
public interface Closeable { void close(); }
public interface FancyStream<T> extends Stream<T>, Closeable { void reset(); }
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type FancyStream[T any] interface { Stream[T] Closeable") {
		t.Fatalf("expected FancyStream to embed both Stream[T] and Closeable, got:\n%s", out)
	}
	if strings.Contains(flat, "*Stream") || strings.Contains(flat, "*Closeable") {
		t.Fatalf("expected embedded interfaces without pointer indirection, got:\n%s", out)
	}
}
