package transpiler

import "testing"

// Execute the emitted enum implementation and the original printAll method.
// The original Java declarations remain in the integration parent unchanged.
func enumNameOriginal191AssertSemantics(t *testing.T, generated string) {
	t.Helper()
	runGoTestInTempModule(t, generated, `
package main

import (
    "fmt"
    "io"
    "os"
    "reflect"
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

var _ fmt.Stringer = (*Day)(nil)

func TestEnumNameOriginal191Behavior(t *testing.T) {
    execution := stdjava.NewExecution()
    names := []struct { host string; units []uint16 }{
        {"MON", []uint16{77, 79, 78}},
        {"FRI", []uint16{70, 82, 73}},
    }
    for _, want := range names {
        day := DayValueOf(stdjava.JavaStringLiteralUTF16(want.units))
        name := day.Name()
        if name == nil || !reflect.DeepEqual(name.UTF16Copy(), want.units) {
            t.Fatalf("%s name UTF16 = %v", want.host, name)
        }
        if day.StringJava2goExecution(execution) != name || stdjava.JavaStringValueOfExecution(execution, day) != name {
            t.Fatalf("%s default string conversion lost the Java name reference", want.host)
        }
        if got := day.String(); got != want.host { t.Fatalf("String()=%q, want %q", got, want.host) }
        if got := fmt.Sprint(day); got != want.host { t.Fatalf("fmt text=%q, want %q", got, want.host) }
    }
    var absent *Day
    if absent.String() != "null" || fmt.Sprint(absent) != "null" { t.Fatal("nil host string bridge must produce null") }
    nullUnits := []uint16{110, 117, 108, 108}
    for _, got := range []*stdjava.JavaString{
        stdjava.JavaStringValueOfExecution(execution, absent),
        stdjava.JavaStringTextOperandExecution(execution, absent),
    } {
        if got == nil || !reflect.DeepEqual(got.UTF16Copy(), nullUnits) { t.Fatalf("nil Java conversion=%v", got) }
    }
    for label, invoke := range map[string]func(){
        "toString": func() { absent.StringJava2goExecution(execution) },
        "name": func() { absent.Name() },
    } {
        func() {
            defer func() {
                recovered := recover()
                _, throwable := recovered.(stdjava.Throwable)
                if !throwable || !stdjava.CaughtAs(recovered, "NullPointerException") {
                    t.Fatalf("nil %s panic=%#v, want canonical Java NullPointerException", label, recovered)
                }
            }()
            invoke()
            t.Errorf("nil %s returned", label)
        }()
    }
    read, write, err := os.Pipe()
    if err != nil { t.Fatal(err) }
    previous := os.Stdout
    defer func() { os.Stdout = previous; _ = read.Close(); _ = write.Close() }()
    os.Stdout = write
    PrintAll()
    os.Stdout = previous
    if err := write.Close(); err != nil { t.Fatal(err) }
    got, err := io.ReadAll(read)
    if err != nil { t.Fatal(err) }
    // Fresh JDK21 observation of the unchanged original EnumText.printAll.
    const want = "FRI\nday=FRI\nFRI\n"
    if string(got) != want { t.Fatalf("original printAll stdout=%q, want %q", got, want) }
}
`)
}
