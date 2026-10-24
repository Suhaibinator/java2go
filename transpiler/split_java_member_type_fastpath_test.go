package transpiler

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

var (
	splitMemberBaseSink      string
	splitMemberArgumentsSink []string
	splitMemberOKSink        bool
)

func TestSplitJavaMemberTypePlainQueriesNoAllocations(t *testing.T) {
	queries := []string{
		"PoolThreadCache",
		"io.netty.buffer.PoolArena",
		"pkg.Outer$Inner[]",
		"Outer.Inner[][]",
		" \tjava.lang.String\r\n",
		"\u2003名.型[]\u00a0",
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			allocations := testing.AllocsPerRun(1000, func() {
				splitMemberBaseSink, splitMemberArgumentsSink, splitMemberOKSink = splitJavaMemberType(query)
			})
			if splitMemberBaseSink != strings.TrimSpace(query) || splitMemberArgumentsSink != nil || !splitMemberOKSink {
				t.Fatalf("plain query = %q, %#v, %v", splitMemberBaseSink, splitMemberArgumentsSink, splitMemberOKSink)
			}
			if allocations != 0 {
				t.Fatalf("plain compiler query allocated %.0f times per call; want zero", allocations)
			}
		})
	}
}

func TestSplitJavaMemberTypeResultsWarningsAndWrapper(t *testing.T) {
	const tooMany = "Unbalanced angle brackets in type string: too many '>'"
	const unclosed = "Unbalanced angle brackets in type string: unclosed '<'"
	cases := []struct {
		name, input, base string
		arguments         []string
		ok                bool
		warning           string
	}{
		{name: "empty", ok: true},
		{name: "whitespace", input: " \t\r\n\u2003", ok: true},
		{name: "plain", input: "Plain", base: "Plain", ok: true},
		{name: "qualified", input: "java.lang.String", base: "java.lang.String", ok: true},
		{name: "member_arrays", input: "pkg.Outer$Inner[][]", base: "pkg.Outer$Inner[][]", ok: true},
		{name: "trim", input: " \tjava.lang.String\r\n", base: "java.lang.String", ok: true},
		{name: "internal_whitespace", input: " A . B [] ", base: "A . B []", ok: true},
		{name: "unicode_trim", input: "\u2003名.型[]\u00a0", base: "名.型[]", ok: true},
		{name: "zero_width_preserved", input: "Foo\u200b", base: "Foo\u200b", ok: true},
		{name: "fullwidth_angles_preserved", input: "名＜型＞", base: "名＜型＞", ok: true},
		{name: "invalid_utf8_preserved", input: "\xffFoo\xfe", base: "\xffFoo\xfe", ok: true},
		{name: "non_java_spelling_preserved", input: "A,B?*", base: "A,B?*", ok: true},
		{name: "stray_close", input: ">", warning: tooMany},
		{name: "trailing_close", input: "A>", warning: tooMany},
		{name: "member_stray_close", input: "A>.B", warning: tooMany},
		{name: "trimmed_stray_close", input: " \tA>\u2003", warning: tooMany},
		{name: "array_stray_close", input: "A[B>]", warning: tooMany},
		{name: "open", input: "A<", warning: unclosed},
		{name: "open_argument", input: "A<B", warning: unclosed},
		{name: "extra_close", input: "A<B>>", warning: tooMany},
		{name: "discard_partial_arguments", input: "A<B>.C<D", warning: unclosed},
		{name: "close_before_open", input: "A>.B<C", warning: tooMany},
		{name: "simple_generic", input: "List<String>", base: "List", arguments: []string{"String"}, ok: true},
		{name: "member_generics", input: "Outer<String>.Child<Impl>", base: "Outer.Child", arguments: []string{"String", "Impl"}, ok: true},
		{name: "nested_member_generics", input: "pkg.Outer<Map<String, List<Integer>>>.Child<Impl>", base: "pkg.Outer.Child", arguments: []string{"Map<String, List<Integer>>", "Impl"}, ok: true},
		{name: "generic_array", input: " \tA<B>.C<D>[]\u2003", base: "A.C[]", arguments: []string{"B", "D"}, ok: true},
		{name: "unicode_generics", input: "外<型>.内<値>", base: "外.内", arguments: []string{"型", "値"}, ok: true},
		{name: "empty_group", input: "<>", ok: true},
		{name: "empty_member_group", input: "A<>.B", base: "A.B", ok: true},
		{name: "empty_fields", input: "A<,>", base: "A", ok: true},
		{name: "skip_empty_argument_fields", input: "A<B,, C>", base: "A", arguments: []string{"B", "C"}, ok: true},
		{name: "nested_commas", input: "A<Map<X,Y>, Z>", base: "A", arguments: []string{"Map<X,Y>", "Z"}, ok: true},
	}

	logger := log.StandardLogger()
	oldOutput, oldFormatter, oldLevel := logger.Out, logger.Formatter, logger.GetLevel()
	t.Cleanup(func() {
		logger.SetOutput(oldOutput)
		logger.SetFormatter(oldFormatter)
		logger.SetLevel(oldLevel)
	})
	var output bytes.Buffer
	logger.SetOutput(&output)
	logger.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
	logger.SetLevel(log.WarnLevel)
	checkWarning := func(t *testing.T, input, warning string) {
		t.Helper()
		if warning == "" {
			if output.Len() != 0 {
				t.Fatalf("unexpected diagnostic: %s", output.String())
			}
			return
		}
		var entry map[string]any
		if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
			t.Fatalf("expected exactly one JSON warning: %q: %v", output.String(), err)
		}
		if len(entry) != 3 || entry["level"] != "warning" || entry["msg"] != warning || entry["typeStr"] != strings.TrimSpace(input) {
			t.Fatalf("warning = %#v, want exact level/message/trimmed typeStr", entry)
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output.Reset()
			base, arguments, ok := splitJavaMemberType(tc.input)
			if base != tc.base || !reflect.DeepEqual(arguments, tc.arguments) || ok != tc.ok {
				t.Fatalf("split = %q, %#v, %v; want %q, %#v, %v", base, arguments, ok, tc.base, tc.arguments, tc.ok)
			}
			checkWarning(t, tc.input, tc.warning)
			output.Reset()
			base, arguments = parseJavaTypeString(tc.input)
			wantBase := tc.base
			if !tc.ok {
				wantBase = strings.TrimSpace(tc.input)
			}
			if base != wantBase || !reflect.DeepEqual(arguments, tc.arguments) {
				t.Fatalf("wrapper = %q, %#v; want %q, %#v", base, arguments, wantBase, tc.arguments)
			}
			checkWarning(t, tc.input, tc.warning)
		})
	}
}
