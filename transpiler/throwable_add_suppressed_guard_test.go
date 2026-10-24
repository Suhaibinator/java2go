package transpiler

import (
	"strings"
	"testing"
)

func TestThrowableAddSuppressedCanonicalDeclarationGuards(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		builtin    bool
	}{
		{"canonical", `class Probe {void run(java.security.GeneralSecurityException value,Throwable other){value.addSuppressed(other);}}`, true},
		{"source inherited", `class Child extends java.security.GeneralSecurityException{} class Probe {void run(Child value,Throwable other){value.addSuppressed(other);}}`, true},
		{"source Object overload inherited wins", `class Child extends java.security.GeneralSecurityException{void addSuppressed(Object value){}} class Probe {void run(Child value,Throwable other){value.addSuppressed(other);}}`, true},
		{"source String overload", `class Child extends java.security.GeneralSecurityException{void addSuppressed(String value){}} class Probe {void run(Child value,String other){value.addSuppressed(other);}}`, false},
		{"source Object overload", `class Child extends java.security.GeneralSecurityException{void addSuppressed(Object value){}} class Probe {void run(Child value,Object other){value.addSuppressed(other);}}`, false},
		{"ordinary shadow", `class GeneralSecurityException{void addSuppressed(Throwable other){}} class Probe {void run(GeneralSecurityException value,Throwable other){value.addSuppressed(other);}}`, false},
		{"foreign", `class Probe {void run(foreign.GeneralSecurityException value,Throwable other){value.addSuppressed(other);}}`, false},
		{"unbounded binder", `class Probe {<GeneralSecurityException> void run(GeneralSecurityException value,Throwable other){value.addSuppressed(other);}}`, false},
		{"array", `class Probe {void run(java.security.GeneralSecurityException[] value,Throwable other){value.addSuppressed(other);}}`, false},
		{"source super", `class Child extends java.security.GeneralSecurityException{void run(Throwable other){super.addSuppressed(other);}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, tc.text)
			got := strings.Contains(out, "stdjava.JavaThrowableAddSuppressedExecution(")
			if got != tc.builtin {
				t.Fatalf("canonical final declaration selection=%v want=%v\n%s", got, tc.builtin, out)
			}
		})
	}
}
