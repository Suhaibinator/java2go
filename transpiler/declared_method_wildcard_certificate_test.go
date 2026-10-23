package transpiler

import "testing"

// These pure certificate queries exercise scope boundaries even where the
// unchanged scorer already assigns cost1 and therefore bypasses the new hook.
func TestDeclaredMethodWildcardCertificateScope(t *testing.T) {
	for _, row := range []struct {
		name, header, formal, locals, arguments string
		want                                    bool
		staleCaller                             bool
	}{
		{name: "upper_capture_admitted", want: true},
		{name: "lower_strings_admitted", header: "<E>", formal: "Collection<? super E> values", locals: "List<String> values=null;", want: true},
		{name: "lower_objects_admitted", header: "<E>", formal: "Collection<? super E> values", locals: "List<Object> values=null;", want: true},
		{name: "unbounded_actual_wildcard_refused", header: "<E>", formal: "Collection<? super E> values", locals: "List<?> values=null;"},
		{name: "lower_actual_wildcard_refused", header: "<E>", formal: "Collection<? super E> values", locals: "List<? super Integer> values=null;"},
		{name: "upper_capture_to_lower_refused", header: "<E>", formal: "Collection<? super E> values"},
		{name: "same_owner_refused", locals: "Collection<? extends Integer> values=null;"},
		{name: "nested_actual_refused", locals: "List<? extends List<Integer>> values=null;"},
		{name: "incompatible_bound_refused", locals: "List<String> values=null;"},
		{name: "explicit_invocation_arguments_refused", arguments: "<Integer>:values"},
		{name: "two_arguments_refused", arguments: "values,values"},
		{name: "caller_snapshot_stale_refused", staleCaller: true},
	} {
		t.Run(row.name, func(t *testing.T) {
			header, formal, locals, arguments := row.header, row.formal, row.locals, row.arguments
			if header == "" {
				header = "<E extends Number>"
			}
			if formal == "" {
				formal = "Collection<E> values"
			}
			if locals == "" {
				locals = "List<? extends Integer> values=null;"
			}
			if arguments == "" {
				arguments = "values"
			}
			f := wildcardStrictScoreSetup(t, header, formal, locals, arguments, nil)
			wildcardStrictScoreFreshInputs(t, f)
			if len(f.args) == 0 {
				t.Fatal("actual argument occurrence missing")
			}
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(f.args[0], f.ctx, f.source, &actual)
			actual.javaType = typ
			formals := methodFormalOrigins(f.method, f.owner, nil, f.ctx)
			if !known || len(formals) != 1 || f.ctx.currentFile.SourceInputSnapshot().Validate() != nil {
				t.Fatal("original actual/formal/current-input premises absent")
			}
			if row.staleCaller {
				// Preserve the actual original attachment; the caller cannot donate
				// a freshly constructed snapshot around mutated bytes and an old tree.
				input := f.ctx.currentFile.SourceInputSnapshot()
				f.ctx.currentFile.Source = append(append([]byte(nil), f.ctx.currentFile.Source...), ' ')
				f.source = f.ctx.currentFile.Source
				if input.Validate() == nil {
					t.Fatal("original caller attachment did not observe stale input")
				}
			}
			got := declaredMethodWildcardStrictWidening(f.method, f.owner, f.args, actual, formals[0], f.ctx, f.source)
			t.Logf("WILDCARD_CERTIFICATE_SCOPE case=%s admits=%v want=%v", row.name, got, row.want)
			if got != row.want {
				t.Fatalf("certificate scope changed: admits=%v want=%v", got, row.want)
			}
		})
	}
}
