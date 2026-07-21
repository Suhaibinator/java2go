package transpiler

import (
	"strings"
	"testing"
)

func TestTryWithResourcesSuppressedExceptionBehavior(t *testing.T) {
	src := `
public class ResourceSuppressionProgram {
    static String trace = "";

    static class Resource implements AutoCloseable {
        String name;
        boolean fail;

        Resource(String name, boolean fail) {
            this.name = name;
            this.fail = fail;
        }

        public void close() {
            trace = trace + name;
            if (fail) {
                throw new IllegalStateException(name);
            }
        }
    }

    static class SharedException extends RuntimeException {
        SharedException(String message) {
            super(message);
        }
    }

    static class SameResource implements AutoCloseable {
        SharedException exception;

        SameResource(SharedException exception) {
            this.exception = exception;
        }

        public void close() {
            throw exception;
        }
    }

    public static String bodyAndClose() {
        trace = "";
        try (Resource first = new Resource("1", true);
             Resource second = new Resource("2", true)) {
            trace = trace + "B";
            throw new IllegalArgumentException("body");
        } catch (IllegalArgumentException ex) {
            return trace + ":" + ex.getMessage() + ":" + ex.getSuppressed().length;
        } catch (IllegalStateException ex) {
            return "wrong-close-primary";
        }
    }

    public static String closeOnly() {
        trace = "";
        try (Resource first = new Resource("1", true);
             Resource second = new Resource("2", true)) {
            trace = trace + "B";
        } catch (IllegalStateException ex) {
            return trace + ":" + ex.getMessage() + ":" + ex.getSuppressed().length;
        }
        return "missing-close";
    }

    public static String returnVsClose() {
        trace = "";
        try (Resource resource = new Resource("C", true)) {
            trace = trace + "B";
            return "wrong-return";
        } catch (IllegalStateException ex) {
            return trace + ":" + ex.getMessage();
        }
    }

    public static String selfSuppression() {
        SharedException shared = new SharedException("same");
        try (SameResource resource = new SameResource(shared)) {
            throw shared;
        } catch (IllegalArgumentException ex) {
            return ex.getMessage() + ":" + ex.getSuppressed().length + ":" +
                   (ex.getCause() == shared) + ":" + ex.getCause().getMessage();
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.CloseResource(func()") {
		t.Fatalf("expected suppression-aware resource cleanup:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.GetSuppressed(ex)") {
		t.Fatalf("expected Throwable.getSuppressed bridge:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.GetCause(ex)") {
		t.Fatalf("expected Throwable.getCause bridge:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestBodyAndClose(t *testing.T) {
	if got := BodyAndClose(); got != "B21:body:2" {
		t.Fatalf("BodyAndClose() = %q, want %q", got, "B21:body:2")
	}
}

func TestCloseOnly(t *testing.T) {
	if got := CloseOnly(); got != "B21:2:1" {
		t.Fatalf("CloseOnly() = %q, want %q", got, "B21:2:1")
	}
}

func TestReturnVsClose(t *testing.T) {
	if got := ReturnVsClose(); got != "BC:C" {
		t.Fatalf("ReturnVsClose() = %q, want %q", got, "BC:C")
	}
}

func TestSelfSuppression(t *testing.T) {
	if got := SelfSuppression(); got != "Self-suppression not permitted:0:true:same" {
		t.Fatalf("SelfSuppression() = %q, want %q", got, "Self-suppression not permitted:0:true:same")
	}
}
`)
}
