package starlarkgen

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

func Test_expr(t *testing.T) {
	tests := []struct {
		name string

		inputBinary    *syntax.BinaryExpr
		inputCallExpr  *syntax.CallExpr
		inputComp      *syntax.Comprehension
		inputCondExpr  *syntax.CondExpr
		inputDictEntry *syntax.DictEntry
		inputDictExpr  *syntax.DictExpr
		inputDotExpr   *syntax.DotExpr
		inputIdent     *syntax.Ident
		inputIndexExpr *syntax.IndexExpr
		inputListExpr  *syntax.ListExpr
		inputLiteral   *syntax.Literal
		inputParen     *syntax.ParenExpr
		inputSliceExpr *syntax.SliceExpr
		inputTupleExpr *syntax.TupleExpr
		inputUnaryExpr *syntax.UnaryExpr

		opts []Option

		want    string
		wantErr string
	}{
		{
			name:        "binary",
			inputBinary: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Y: &syntax.Literal{Value: "bar"}, Op: syntax.EQL},
			want:        `foo == "bar"`,
		},
		{
			name:        "binary, special case Op == EQ",
			inputBinary: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Y: &syntax.Literal{Value: "bar"}, Op: syntax.EQ},
			want:        `foo="bar"`,
		},
		{
			name:        "binary, special case Op == EQ, whitespace added with options",
			inputBinary: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Y: &syntax.Literal{Value: "bar"}, Op: syntax.EQ},
			opts:        []Option{WithSpaceEqBinary(true)},
			want:        `foo = "bar"`,
		},
		{
			name:          "call, empty arg list",
			inputCallExpr: &syntax.CallExpr{Fn: &syntax.Ident{Name: "foo"}},
			want:          "foo()",
		},
		{
			name:          "call, single argument",
			inputCallExpr: &syntax.CallExpr{Fn: &syntax.Ident{Name: "foo"}, Args: []syntax.Expr{&syntax.Ident{Name: "bar"}}},
			want:          "foo(bar)",
		},
		{
			name: "call, multiple arguments",
			inputCallExpr: &syntax.CallExpr{Fn: &syntax.Ident{Name: "foo"}, Args: []syntax.Expr{
				&syntax.Ident{Name: "bar"},
				&syntax.Literal{Value: 2},
				&syntax.Literal{Value: "foobar"}},
			},
			want: `foo(bar, 2, "foobar")`,
		},
		{
			name: "comprehension, list with for and if",
			inputComp: &syntax.Comprehension{
				Body: &syntax.Ident{Name: "x"},
				Clauses: []syntax.Node{
					&syntax.ForClause{
						X:    &syntax.Ident{Name: "y"},
						Vars: &syntax.Ident{Name: "x"},
					},
					&syntax.IfClause{
						Cond: &syntax.BinaryExpr{
							Op: syntax.LT,
							X:  &syntax.Ident{Name: "x"},
							Y:  &syntax.Literal{Value: 10},
						},
					},
				},
			},
			want: "[x for x in y if x < 10]",
		},
		{
			name: "comprehension, dict with for and if",
			inputComp: &syntax.Comprehension{
				Curly: true,
				Body:  &syntax.DictEntry{Key: &syntax.Ident{Name: "x"}, Value: &syntax.Ident{Name: "y"}},
				Clauses: []syntax.Node{
					&syntax.ForClause{
						X:    &syntax.Ident{Name: "z"},
						Vars: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Ident{Name: "x"}, &syntax.Ident{Name: "y"}}},
					},
					&syntax.IfClause{
						Cond: &syntax.BinaryExpr{
							Op: syntax.LT,
							X:  &syntax.Ident{Name: "x"},
							Y:  &syntax.Literal{Value: 10},
						},
					},
				},
			},
			want: "{x: y for x, y in z if x < 10}",
		},
		{
			name: "comprehension, failure, unsupported clause",
			inputComp: &syntax.Comprehension{
				Body: &syntax.Ident{Name: "x"},
				Clauses: []syntax.Node{
					&syntax.Literal{},
				},
			},
			wantErr: "unexpected clause type *syntax.Literal rendering comprehension",
		},
		{
			name: "condition expression",
			inputCondExpr: &syntax.CondExpr{
				Cond:  &syntax.BinaryExpr{Op: syntax.LT, X: &syntax.Ident{Name: "foo"}, Y: &syntax.Ident{Name: "bar"}},
				True:  &syntax.UnaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.MINUS},
				False: &syntax.BinaryExpr{Op: syntax.STAR, X: &syntax.Ident{Name: "bar"}, Y: &syntax.Literal{Value: 2}},
			},
			want: "-foo if foo < bar else bar * 2",
		},
		{
			name:           "dict entry",
			inputDictEntry: &syntax.DictEntry{Key: &syntax.Literal{Value: "foo"}, Value: &syntax.Literal{Value: "bar"}},
			want:           `"foo": "bar"`,
		},
		{
			name:          "dict, empty",
			inputDictExpr: &syntax.DictExpr{},
			want:          "{}",
		},
		{
			name:          "dict, single key value pair",
			inputDictExpr: &syntax.DictExpr{List: []syntax.Expr{&syntax.DictEntry{Key: &syntax.Literal{Value: "foo"}, Value: &syntax.Literal{Value: 2}}}},
			want:          `{"foo": 2}`,
		},
		{
			name: "dict, multiple key value pairs",
			inputDictExpr: &syntax.DictExpr{List: []syntax.Expr{
				&syntax.DictEntry{Key: &syntax.Literal{Value: "foo"}, Value: &syntax.Literal{Value: 2}},
				&syntax.DictEntry{Key: &syntax.Literal{Value: "bar"}, Value: &syntax.Literal{Value: "z"}},
				&syntax.DictEntry{Key: &syntax.Literal{Value: "foobar"}, Value: &syntax.Ident{Name: "some_var"}},
			}},
			want: `{"foo": 2, "bar": "z", "foobar": some_var}`,
		},
		{
			name:          "dict, invalid contents",
			inputDictExpr: &syntax.DictExpr{List: []syntax.Expr{&syntax.Literal{Value: "foo"}}},
			wantErr:       "expected *syntax.DictEntry, got *syntax.Literal in dictExpr",
		},
		{
			name:         "dot expr, simple ident.ident",
			inputDotExpr: &syntax.DotExpr{X: &syntax.Ident{Name: "foo"}, Name: &syntax.Ident{Name: "bar"}},
			want:         "foo.bar",
		},
		{
			name:         "dot expr, literal function call",
			inputDotExpr: &syntax.DotExpr{X: &syntax.Literal{Value: "foo"}, Name: &syntax.Ident{Name: "format"}},
			want:         `"foo".format`,
		},
		{
			name:       "ident",
			inputIdent: &syntax.Ident{Name: "foobar"},
			want:       "foobar",
		},
		{
			name:           "index expression, integer index",
			inputIndexExpr: &syntax.IndexExpr{X: &syntax.Ident{Name: "foo"}, Y: &syntax.Literal{Value: 2}},
			want:           "foo[2]",
		},
		{
			name:          "list expression, empty list",
			inputListExpr: &syntax.ListExpr{},
			want:          "[]",
		},
		{
			name:          "list expression, single element list",
			inputListExpr: &syntax.ListExpr{List: []syntax.Expr{&syntax.Ident{Name: "foo"}}},
			want:          "[foo]",
		},
		{
			name:          "list expression, multiple elements list",
			inputListExpr: &syntax.ListExpr{List: []syntax.Expr{&syntax.Ident{Name: "foo"}, &syntax.Literal{Value: 2}, &syntax.Literal{Value: "bar"}}},
			want:          `[foo, 2, "bar"]`,
		},
		{
			name:         "literal, string value",
			inputLiteral: &syntax.Literal{Value: "foo"},
			want:         `"foo"`,
		},
		{
			name:         "literal, int value",
			inputLiteral: &syntax.Literal{Value: -2},
			want:         "-2",
		},
		{
			name:         "literal, uint value",
			inputLiteral: &syntax.Literal{Value: uint(2)},
			want:         "2",
		},
		{
			name:         "literal, int64 value",
			inputLiteral: &syntax.Literal{Value: int64(-2)},
			want:         "-2",
		},
		{
			name:         "literal, uint64 value",
			inputLiteral: &syntax.Literal{Value: uint64(2)},
			want:         "2",
		},
		{
			name: "literal, bigint",
			inputLiteral: &syntax.Literal{Value: func() *big.Int {
				v, _ := new(big.Int).SetString("9999999999999999999", 10) // that's the largest number I know
				return v
			}()},
			want: "9999999999999999999",
		},
		{
			name:         "literal, nil bigint",
			inputLiteral: &syntax.Literal{Value: (*big.Int)(nil)},
			wantErr:      "nil literal *big.Int value provided",
		},
		{
			name:         "literal, unsupported value type",
			inputLiteral: &syntax.Literal{Value: struct{}{}},
			wantErr:      "unsupported literal value type struct {}, expected string, int, int64, uint, uint64 or *big.Int",
		},
		{
			name:         "literal, raw value",
			inputLiteral: &syntax.Literal{Raw: " foo ---"},
			want:         " foo ---",
		},
		{
			name:       "parenExpr, variable",
			inputParen: &syntax.ParenExpr{X: &syntax.Ident{Name: "foo"}},
			want:       "(foo)",
		},
		{
			name:       "parenExpr, tuple pair",
			inputParen: &syntax.ParenExpr{X: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Ident{Name: "foo"}, &syntax.Ident{Name: "bar"}}}},
			want:       "(foo, bar)",
		},
		{
			name:           "slice expression, empty ranges",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}},
			want:           "foo[:]",
		},
		{
			name:           "slice expression, Hi range set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Hi: &syntax.Literal{Value: 2}},
			want:           "foo[:2]",
		},
		{
			name:           "slice expression, Step range set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Step: &syntax.Literal{Value: 3}},
			want:           "foo[::3]",
		},
		{
			name:           "slice expression, Lo range set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Lo: &syntax.Ident{Name: "lo"}},
			want:           "foo[lo:]",
		},
		{
			name:           "slice expression, Lo and Step ranges set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Lo: &syntax.Ident{Name: "lo"}, Step: &syntax.Literal{Value: 3}},
			want:           "foo[lo::3]",
		},
		{
			name:           "slice expression, Lo and Hi ranges set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Lo: &syntax.Ident{Name: "lo"}, Hi: &syntax.Literal{Value: 2}},
			want:           "foo[lo:2]",
		},
		{
			name:           "slice expression, Hi and Step ranges set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Hi: &syntax.Literal{Value: 2}, Step: &syntax.Literal{Value: 3}},
			want:           "foo[:2:3]",
		},
		{
			name:           "slice expression, Lo, Hi and Step ranges set",
			inputSliceExpr: &syntax.SliceExpr{X: &syntax.Ident{Name: "foo"}, Lo: &syntax.Ident{Name: "lo"}, Hi: &syntax.Literal{Value: 2}, Step: &syntax.Literal{Value: 3}},
			want:           "foo[lo:2:3]",
		},
		{
			name:           "tuple, pair",
			inputTupleExpr: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Literal{Value: 1}, &syntax.Literal{Value: 2}}},
			want:           "1, 2",
		},
		{
			name:           "tuple, multiple",
			inputTupleExpr: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Literal{Value: 1}, &syntax.Literal{Value: 2}, &syntax.Ident{Name: "foo"}}},
			want:           "1, 2, foo",
		},
		{
			name:           "unary expr",
			inputUnaryExpr: &syntax.UnaryExpr{Op: syntax.MINUS, X: &syntax.Ident{Name: "foo"}},
			want:           "-foo",
		},
		{
			name:           "unary expr, special case single star",
			inputUnaryExpr: &syntax.UnaryExpr{Op: syntax.STAR},
			want:           "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				sb           strings.Builder
				exprSb       strings.Builder
				err          error
				errExpr      error
				inputExpr    syntax.Expr
				opts, optErr = getOutputOpts(tt.opts...)
			)
			if optErr != nil {
				t.Fatalf("invalid options: %v", optErr)
			}

			switch {
			case tt.inputBinary != nil:
				err = binaryExpr(&sb, tt.inputBinary, opts)
				inputExpr = tt.inputBinary
			case tt.inputCallExpr != nil:
				err = callExpr(&sb, tt.inputCallExpr, opts)
				inputExpr = tt.inputCallExpr
			case tt.inputComp != nil:
				err = comprehension(&sb, tt.inputComp, opts)
				inputExpr = tt.inputComp
			case tt.inputCondExpr != nil:
				err = condExpr(&sb, tt.inputCondExpr, opts)
				inputExpr = tt.inputCondExpr
			case tt.inputDictEntry != nil:
				err = dictEntry(&sb, tt.inputDictEntry, opts)
				inputExpr = tt.inputDictEntry
			case tt.inputDictExpr != nil:
				err = dictExpr(&sb, tt.inputDictExpr, opts)
				inputExpr = tt.inputDictExpr
			case tt.inputDotExpr != nil:
				err = dotExpr(&sb, tt.inputDotExpr, opts)
				inputExpr = tt.inputDotExpr
			case tt.inputIdent != nil:
				err = ident(&sb, tt.inputIdent, opts)
				inputExpr = tt.inputIdent
			case tt.inputIndexExpr != nil:
				err = indexExpr(&sb, tt.inputIndexExpr, opts)
				inputExpr = tt.inputIndexExpr
			case tt.inputListExpr != nil:
				err = listExpr(&sb, tt.inputListExpr, opts)
				inputExpr = tt.inputListExpr
			case tt.inputLiteral != nil:
				err = literal(&sb, tt.inputLiteral, opts)
				inputExpr = tt.inputLiteral
			case tt.inputParen != nil:
				err = parenExpr(&sb, tt.inputParen, opts)
				inputExpr = tt.inputParen
			case tt.inputSliceExpr != nil:
				err = sliceExpr(&sb, tt.inputSliceExpr, opts)
				inputExpr = tt.inputSliceExpr
			case tt.inputTupleExpr != nil:
				err = tupleExpr(&sb, tt.inputTupleExpr, opts)
				inputExpr = tt.inputTupleExpr
			case tt.inputUnaryExpr != nil:
				err = unaryExpr(&sb, tt.inputUnaryExpr, opts)
				inputExpr = tt.inputUnaryExpr
			default:
				t.Fatal("test value not set")
			}
			// check if expr() provides same results as type-specific functions
			errExpr = expr(&exprSb, inputExpr, opts)

			if tt.wantErr != "" {
				if err == nil || errExpr == nil {
					t.Fatal("expected an error, got nil")
				}
				if gotErr, gotExprErr := err.Error(), errExpr.Error(); gotErr != tt.wantErr || gotExprErr != tt.wantErr {
					t.Errorf("expected error %q, got %q and %q from expr()", tt.wantErr, gotErr, gotExprErr)
				}
				return
			}
			if err != nil || errExpr != nil {
				t.Fatalf("expected no error, got %v and %v from expr()", err, errExpr)
			}
			if got, gotExpr := sb.String(), exprSb.String(); got != tt.want || gotExpr != tt.want {
				t.Errorf("expected %q, got %q and %q from expr()", tt.want, got, gotExpr)
			}
		})
	}
}

func Test_WithCallOption_invalid(t *testing.T) {
	tests := []CallOption{
		callOptionMax,
		callOptionMax + 1,
		CallOption(0xff),
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("List option %v failure", tt), func(t *testing.T) {
			if opts, err := getOutputOpts(WithCallOption(tt)); opts != nil || err == nil {
				t.Errorf("expected nil options and error, got %v and %v", opts, err)
			}
		})
	}
}
func Test_WithDictOption_invalid(t *testing.T) {
	tests := []DictOption{
		dictOptionMax,
		dictOptionMax + 1,
		DictOption(0xff),
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Dict option %v failure", tt), func(t *testing.T) {
			if opts, err := getOutputOpts(WithDictOption(tt)); opts != nil || err == nil {
				t.Errorf("expected nil options and error, got %v and %v", opts, err)
			}
		})
	}
}

func Test_WithListOption_invalid(t *testing.T) {
	tests := []ListOption{
		listOptionMax,
		listOptionMax + 1,
		ListOption(0xff),
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("List option %v failure", tt), func(t *testing.T) {
			if opts, err := getOutputOpts(WithListOption(tt)); opts != nil || err == nil {
				t.Errorf("expected nil options and error, got %v and %v", opts, err)
			}
		})
	}
}

func Test_WithTupleOption_invalid(t *testing.T) {
	tests := []TupleOption{
		tupleOptionMax,
		tupleOptionMax + 1,
		TupleOption(0xff),
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("List option %v failure", tt), func(t *testing.T) {
			if opts, err := getOutputOpts(WithTupleOption(tt)); opts != nil || err == nil {
				t.Errorf("expected nil options and error, got %v and %v", opts, err)
			}
		})
	}
}

func Test_withCallOption(t *testing.T) {
	testMatrix := map[string]syntax.Expr{
		"single": &syntax.CallExpr{
			Fn:   &syntax.Ident{Name: "some_func"},
			Args: []syntax.Expr{&syntax.Ident{Name: "foo"}},
		},
		"multi": &syntax.CallExpr{
			Fn: &syntax.Ident{Name: "some_func"},
			Args: []syntax.Expr{
				&syntax.Ident{Name: "foo"},
				&syntax.Literal{Value: 1},
				&syntax.Ident{Name: "bar"},
				&syntax.Literal{Value: 2},
				&syntax.Ident{Name: "test"},
				&syntax.Literal{Value: 3},
			},
		},
	}

	tests := []struct {
		withCallOption CallOption
		want           map[string]string
	}{
		{
			withCallOption: CallOptionSingleLine,
			want: map[string]string{
				"single": "some_func(foo)",
				"multi":  "some_func(foo, 1, bar, 2, test, 3)",
			},
		},
		{
			withCallOption: CallOptionSingleLineComma,
			want: map[string]string{
				"single": "some_func(foo,)",
				"multi":  "some_func(foo, 1, bar, 2, test, 3,)",
			},
		},
		{
			withCallOption: CallOptionSingleLineCommaTwoAndMore,
			want: map[string]string{
				"single": "some_func(foo)",
				"multi":  "some_func(foo, 1, bar, 2, test, 3,)",
			},
		},
		{
			withCallOption: CallOptionMultiline,
			want: map[string]string{
				"single": "some_func(\n++foo\n+)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+)",
			},
		},
		{
			withCallOption: CallOptionMultilineComma,
			want: map[string]string{
				"single": "some_func(\n++foo,\n+)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withCallOption: CallOptionMultilineCommaTwoAndMore,
			want: map[string]string{
				"single": "some_func(\n++foo\n+)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withCallOption: CallOptionMultilineMultiple,
			want: map[string]string{
				"single": "some_func(foo)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+)",
			},
		},
		{
			withCallOption: CallOptionMultilineMultipleComma,
			want: map[string]string{
				"single": "some_func(foo,)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withCallOption: CallOptionMultilineMultipleCommaTwoAndMore,
			want: map[string]string{
				"single": "some_func(foo)",
				"multi":  "some_func(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
	}
	for _, tt := range tests {
		for name, value := range testMatrix {
			opts := []Option{WithCallOption(tt.withCallOption), WithIndent("+"), WithDepth(1)}
			t.Run(fmt.Sprintf("empty, multiline: %#v", tt.withCallOption), func(t *testing.T) {
				// empty call is always rendered as "some_func()"
				if got, err := StarlarkExpr(&syntax.CallExpr{Fn: &syntax.Ident{Name: "some_func"}}, opts...); err != nil || got != "some_func()" {
					t.Errorf("expected nil error and \"some_func()\", got %v and %q", err, got)
				}
			})
			t.Run(fmt.Sprintf("%v, multiline: %#v", name, tt.withCallOption), func(t *testing.T) {
				if got, err := StarlarkExpr(value, opts...); err != nil || got != tt.want[name] {
					t.Errorf("expected nil error and %q, got %v and %q", tt.want[name], err, got)
				}
			})
		}
	}
}

func Test_withDictOption(t *testing.T) {
	testMatrix := map[string]syntax.Expr{
		"single": &syntax.DictExpr{List: []syntax.Expr{&syntax.DictEntry{Key: &syntax.Ident{Name: "foo"}, Value: &syntax.Literal{Value: 1}}}},
		"multi": &syntax.DictExpr{List: []syntax.Expr{
			&syntax.DictEntry{Key: &syntax.Ident{Name: "foo"}, Value: &syntax.Literal{Value: 1}},
			&syntax.DictEntry{Key: &syntax.Ident{Name: "bar"}, Value: &syntax.Literal{Value: 2}},
			&syntax.DictEntry{Key: &syntax.Ident{Name: "test"}, Value: &syntax.Literal{Value: 3}},
		}},
	}

	tests := []struct {
		withDictOption DictOption
		want           map[string]string
	}{
		{
			withDictOption: DictOptionSingleLine,
			want: map[string]string{
				"single": "{foo: 1}",
				"multi":  "{foo: 1, bar: 2, test: 3}",
			},
		},
		{
			withDictOption: DictOptionSingleLineComma,
			want: map[string]string{
				"single": "{foo: 1,}",
				"multi":  "{foo: 1, bar: 2, test: 3,}",
			},
		},
		{
			withDictOption: DictOptionSingleLineCommaTwoAndMore,
			want: map[string]string{
				"single": "{foo: 1}",
				"multi":  "{foo: 1, bar: 2, test: 3,}",
			},
		},
		{
			withDictOption: DictOptionMultiline,
			want: map[string]string{
				"single": "{\n++foo: 1\n+}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3\n+}",
			},
		},
		{
			withDictOption: DictOptionMultilineComma,
			want: map[string]string{
				"single": "{\n++foo: 1,\n+}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3,\n+}",
			},
		},
		{
			withDictOption: DictOptionMultilineCommaTwoAndMore,
			want: map[string]string{
				"single": "{\n++foo: 1\n+}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3,\n+}",
			},
		},
		{
			withDictOption: DictOptionMultilineMultiple,
			want: map[string]string{
				"single": "{foo: 1}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3\n+}",
			},
		},
		{
			withDictOption: DictOptionMultilineMultipleComma,
			want: map[string]string{
				"single": "{foo: 1,}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3,\n+}",
			},
		},
		{
			withDictOption: DictOptionMultilineMultipleCommaTwoAndMore,
			want: map[string]string{
				"single": "{foo: 1}",
				"multi":  "{\n++foo: 1,\n++bar: 2,\n++test: 3,\n+}",
			},
		},
	}
	for _, tt := range tests {
		for name, value := range testMatrix {
			opts := []Option{WithDictOption(tt.withDictOption), WithIndent("+"), WithDepth(1)}
			t.Run(fmt.Sprintf("empty, multiline: %#v", tt.withDictOption), func(t *testing.T) {
				// empty dict is always rendered as "{}"
				if got, err := StarlarkExpr(&syntax.DictExpr{}, opts...); err != nil || got != "{}" {
					t.Errorf("expected nil error and \"{}\", got %v and %q", err, got)
				}
			})
			t.Run(fmt.Sprintf("%v, multiline: %#v", name, tt.withDictOption), func(t *testing.T) {
				if got, err := StarlarkExpr(value, opts...); err != nil || got != tt.want[name] {
					t.Errorf("expected nil error and %q, got %v and %q", tt.want[name], err, got)
				}
			})
		}
	}
}

func Test_withListOption(t *testing.T) {
	testMatrix := map[string]syntax.Expr{
		"single": &syntax.ListExpr{List: []syntax.Expr{&syntax.Ident{Name: "foo"}}},
		"multi": &syntax.ListExpr{List: []syntax.Expr{
			&syntax.Ident{Name: "foo"},
			&syntax.Literal{Value: 1},
			&syntax.Ident{Name: "bar"},
			&syntax.Literal{Value: 2},
			&syntax.Ident{Name: "test"},
			&syntax.Literal{Value: 3},
		}},
	}

	tests := []struct {
		withListOption ListOption
		want           map[string]string
	}{
		{
			withListOption: ListOptionSingleLine,
			want: map[string]string{
				"single": "[foo]",
				"multi":  "[foo, 1, bar, 2, test, 3]",
			},
		},
		{
			withListOption: ListOptionSingleLineComma,
			want: map[string]string{
				"single": "[foo,]",
				"multi":  "[foo, 1, bar, 2, test, 3,]",
			},
		},
		{
			withListOption: ListOptionSingleLineCommaTwoAndMore,
			want: map[string]string{
				"single": "[foo]",
				"multi":  "[foo, 1, bar, 2, test, 3,]",
			},
		},
		{
			withListOption: ListOptionMultiline,
			want: map[string]string{
				"single": "[\n++foo\n+]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+]",
			},
		},
		{
			withListOption: ListOptionMultilineComma,
			want: map[string]string{
				"single": "[\n++foo,\n+]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+]",
			},
		},
		{
			withListOption: ListOptionMultilineCommaTwoAndMore,
			want: map[string]string{
				"single": "[\n++foo\n+]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+]",
			},
		},
		{
			withListOption: ListOptionMultilineMultiple,
			want: map[string]string{
				"single": "[foo]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+]",
			},
		},
		{
			withListOption: ListOptionMultilineMultipleComma,
			want: map[string]string{
				"single": "[foo,]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+]",
			},
		},
		{
			withListOption: ListOptionMultilineMultipleCommaTwoAndMore,
			want: map[string]string{
				"single": "[foo]",
				"multi":  "[\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+]",
			},
		},
	}
	for _, tt := range tests {
		for name, value := range testMatrix {
			opts := []Option{WithListOption(tt.withListOption), WithIndent("+"), WithDepth(1)}
			t.Run(fmt.Sprintf("empty, multiline: %#v", tt.withListOption), func(t *testing.T) {
				// empty list is always rendered as "[]"
				if got, err := StarlarkExpr(&syntax.ListExpr{}, opts...); err != nil || got != "[]" {
					t.Errorf("expected nil error and \"[]\", got %v and %q", err, got)
				}
			})
			t.Run(fmt.Sprintf("%v, multiline: %#v", name, tt.withListOption), func(t *testing.T) {
				if got, err := StarlarkExpr(value, opts...); err != nil || got != tt.want[name] {
					t.Errorf("expected nil error and %q, got %v and %q", tt.want[name], err, got)
				}
			})
		}
	}
}

func Test_withTupleOption(t *testing.T) {
	testMatrix := map[string]syntax.Expr{
		"single": &syntax.ParenExpr{X: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Ident{Name: "foo"}}}},
		"multi": &syntax.ParenExpr{
			X: &syntax.TupleExpr{List: []syntax.Expr{
				&syntax.Ident{Name: "foo"},
				&syntax.Literal{Value: 1},
				&syntax.Ident{Name: "bar"},
				&syntax.Literal{Value: 2},
				&syntax.Ident{Name: "test"},
				&syntax.Literal{Value: 3},
			}},
		},
	}

	tests := []struct {
		withTupleOption TupleOption
		want            map[string]string
	}{
		{
			withTupleOption: TupleOptionSingleLine,
			want: map[string]string{
				"single": "(foo)",
				"multi":  "(foo, 1, bar, 2, test, 3)",
			},
		},
		{
			withTupleOption: TupleOptionSingleLineComma,
			want: map[string]string{
				"single": "(foo,)",
				"multi":  "(foo, 1, bar, 2, test, 3,)",
			},
		},
		{
			withTupleOption: TupleOptionSingleLineCommaTwoAndMore,
			want: map[string]string{
				"single": "(foo)",
				"multi":  "(foo, 1, bar, 2, test, 3,)",
			},
		},
		{
			withTupleOption: TupleOptionMultiline,
			want: map[string]string{
				"single": "(\n++foo\n+)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+)",
			},
		},
		{
			withTupleOption: TupleOptionMultilineComma,
			want: map[string]string{
				"single": "(\n++foo,\n+)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withTupleOption: TupleOptionMultilineCommaTwoAndMore,
			want: map[string]string{
				"single": "(\n++foo\n+)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withTupleOption: TupleOptionMultilineMultiple,
			want: map[string]string{
				"single": "(foo)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3\n+)",
			},
		},
		{
			withTupleOption: TupleOptionMultilineMultipleComma,
			want: map[string]string{
				"single": "(foo,)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
		{
			withTupleOption: TupleOptionMultilineMultipleCommaTwoAndMore,
			want: map[string]string{
				"single": "(foo)",
				"multi":  "(\n++foo,\n++1,\n++bar,\n++2,\n++test,\n++3,\n+)",
			},
		},
	}
	for _, tt := range tests {
		for name, value := range testMatrix {
			opts := []Option{WithTupleOption(tt.withTupleOption), WithIndent("+"), WithDepth(1)}
			t.Run(fmt.Sprintf("empty, multiline: %#v", tt.withTupleOption), func(t *testing.T) {
				// empty tuple is always rendered as "()"
				if got, err := StarlarkExpr(&syntax.ParenExpr{X: &syntax.TupleExpr{}}, opts...); err != nil || got != "()" {
					t.Errorf("expected nil error and \"()\", got %v and %q", err, got)
				}
			})
			t.Run(fmt.Sprintf("%v, multiline: %#v", name, tt.withTupleOption), func(t *testing.T) {
				if got, err := StarlarkExpr(value, opts...); err != nil || got != tt.want[name] {
					t.Errorf("expected nil error and %q, got %v and %q", tt.want[name], err, got)
				}
			})
		}
	}
}

func Test_nilExpr(t *testing.T) {
	tests := []struct {
		name         string
		input        syntax.Expr
		wantErr      string
		wantNonEmpty bool
	}{
		{
			name:    "nil",
			wantErr: "type <nil> is not supported",
		},
		{
			name:    "nil syntax.BinaryExpr",
			input:   (*syntax.BinaryExpr)(nil),
			wantErr: "rendering binary expression: nil input",
		},
		{
			name:    "nil syntax.CallExpr",
			input:   (*syntax.CallExpr)(nil),
			wantErr: "rendering call expression: nil input",
		},
		{
			name:    "nil syntax.Comprehension",
			input:   (*syntax.Comprehension)(nil),
			wantErr: "rendering comprehension: nil input",
		},
		{
			name:    "nil syntax.CondExpr",
			input:   (*syntax.CondExpr)(nil),
			wantErr: "rendering condition expression: nil input",
		},
		{
			name:    "nil syntax.DictEntry",
			input:   (*syntax.DictEntry)(nil),
			wantErr: "rendering dict entry: nil input",
		},
		{
			name:    "nil syntax.DictExpr",
			input:   (*syntax.DictExpr)(nil),
			wantErr: "rendering dict expression: nil input",
		},
		{
			name:    "nil syntax.DotExpr",
			input:   (*syntax.DotExpr)(nil),
			wantErr: "rendering dot expression: nil input",
		},
		{
			name:    "nil syntax.Ident",
			input:   (*syntax.Ident)(nil),
			wantErr: "rendering ident: nil input",
		},
		{
			name:    "nil syntax.IndexExpr",
			input:   (*syntax.IndexExpr)(nil),
			wantErr: "rendering index expression: nil input",
		},
		{
			name:    "nil syntax.ListExpr",
			input:   (*syntax.ListExpr)(nil),
			wantErr: "rendering list expression: nil input",
		},
		{
			name:    "nil syntax.Literal",
			input:   (*syntax.Literal)(nil),
			wantErr: "rendering literal: nil input",
		},
		{
			name:    "nil syntax.ParenExpr",
			input:   (*syntax.ParenExpr)(nil),
			wantErr: "rendering paren expression: nil input",
		},
		{
			name:    "nil syntax.SliceExpr",
			input:   (*syntax.SliceExpr)(nil),
			wantErr: "rendering slice expression: nil input",
		},
		{
			name:    "nil syntax.TupleExpr",
			input:   (*syntax.TupleExpr)(nil),
			wantErr: "rendering tuple expression: nil input",
		},
		{
			name:    "nil syntax.UnaryExpr",
			input:   (*syntax.UnaryExpr)(nil),
			wantErr: "rendering unary expression: nil input",
		},
		{
			name:    "nil syntax.LambdaExpr",
			input:   (*syntax.LambdaExpr)(nil),
			wantErr: "type *syntax.LambdaExpr is not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic, got %v", r)
				}
			}()
			var sb strings.Builder
			err := expr(&sb, tt.input, &defaultOpts)

			if gotText := sb.String(); !tt.wantNonEmpty && gotText != "" {
				t.Fatalf("empty output expected, got %q", gotText)
			}
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if gotErr := err.Error(); gotErr != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, gotErr)
			}
		})
	}
}
