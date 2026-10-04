package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Frozen separately: activate only with the independently owned canonical
// Throwable storage/getter prerequisite. Never observe these units via Message.
func TestCampaignJavaIntegerParseIntMessagesJDK21(t *testing.T) {
	var out strings.Builder
	for _, c := range integerParseReferenceCases() {
		value, thrown := integerParseReferenceRun(c)
		if thrown == nil {
			fmt.Fprintf(&out, "%s=value:%d\n", c.label, value)
			continue
		}
		error, ok := thrown.(NumberFormatException)
		if !ok {
			t.Fatalf("%s threw %T:%v", c.label, thrown, thrown)
		}
		message := JavaThrowableMessageDefault(error)
		fmt.Fprintf(&out, "%s=%s|", c.label, error.ThrowableTypeName())
		if message == nil {
			out.WriteString("null")
		} else {
			for _, unit := range message.UTF16Copy() {
				fmt.Fprintf(&out, "%x,", unit)
			}
		}
		fmt.Fprintf(&out, "|%t\n", message == JavaThrowableMessageDefault(error))
	}
	want, err := os.ReadFile("testdata/integer_parse_messages_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	gotLines, wantLines := strings.Split(out.String(), "\n"), strings.Split(string(want), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("linecount Go%d/JDK%d", len(gotLines), len(wantLines))
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line%d Go:%s JDK21:%s", i+1, gotLines[i], wantLines[i])
		}
	}
}
