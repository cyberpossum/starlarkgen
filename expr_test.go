package starlarkgen

import (
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

type expectingWriter struct {
	expects       map[string]int
	expectedError error
}

func (ew *expectingWriter) WriteString(s string) (int, error) {
	if n, ok := ew.expects[s]; ok {
		if n <= 0 {
			return 0, ew.expectedError
		}
		ew.expects[s]--
	}

	return len(s), nil
}

func newExpectingWriter(token string, failOn int, expected bool) *expectingWriter {
	if failOn <= 0 {
		panic(failOn)
	}
	expectedStr := "AS EXPECTED"
	if !expected {
		expectedStr = "UNEXPECTED"
	}
	return &expectingWriter{
		expects:       map[string]int{token: failOn - 1},
		expectedError: fmt.Errorf("%s: %q occurence %d", expectedStr, token, failOn),
	}
}

type wantSetup struct {
	writerSetup io.StringWriter
	wantErr     string
	opts        []Option
}

func newExpectingWriters(token string, failNum int, errPrefix string, opts ...Option) []wantSetup {
	res := make([]wantSetup, failNum+1)
	for i := 0; i < failNum; i++ {
		res[i] = wantSetup{
			writerSetup: newExpectingWriter(token, i+1, true),
			opts:        opts,
			wantErr:     errPrefix + fmt.Sprintf(" AS EXPECTED: %q occurence %d", token, i+1),
		}
	}
	res[failNum] = wantSetup{
		writerSetup: newExpectingWriter(token, failNum+1, false),
		opts:        opts,
		wantErr:     "",
	}

	return res
}

type nilWriter struct{}

func (n *nilWriter) WriteString(string) (int, error) {
	return 0, nil
}

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
		{
			name:           "unary expr, invalid no X, Op != STAR",
			inputUnaryExpr: &syntax.UnaryExpr{Op: syntax.MINUS},
			wantErr:        "rendering unary expression, nil X value for \"-\" token",
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

func Test_writerFailureExpr(t *testing.T) {
	var (
		fooCond       = &syntax.Ident{Name: "foo_cond"}
		fooIdent      = &syntax.Ident{Name: "foo"}
		xIdent        = &syntax.Ident{Name: "x"}
		yIdent        = &syntax.Ident{Name: "y"}
		oneLiteral    = &syntax.Literal{Value: 1}
		twentyLiteral = &syntax.Literal{Value: 20}
	)
	testMap := map[syntax.Node][][]wantSetup{
		// expressions
		&syntax.BinaryExpr{X: xIdent, Y: yIdent, Op: syntax.EQL}: {
			newExpectingWriters("x", 1, "rendering binary expression X: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering binary expression Y: rendering ident Name:"),
			newExpectingWriters("==", 1, "rendering binary expression Op token:"),
			newExpectingWriters(" ", 2, "rendering binary expression space:"),
		},
		&syntax.CallExpr{Args: []syntax.Expr{oneLiteral, xIdent, twentyLiteral}, Fn: yIdent}: {
			newExpectingWriters("(", 1, "rendering call expression LPAREN token:"),
			newExpectingWriters(")", 1, "rendering call expression RPAREN token:"),
			newExpectingWriters(" ", 2, "rendering call expression space:"),
			newExpectingWriters(",", 2, "rendering call expression COMMA token:"),
			newExpectingWriters(",", 3, "rendering call expression COMMA token:", WithCallOption(CallOptionSingleLineComma)),
			newExpectingWriters("\n", 4, "rendering call expression NEWLINE token:", WithCallOption(CallOptionMultilineComma)),
			newExpectingWriters("    ", 3, "rendering call expression indent:", WithCallOption(CallOptionMultilineComma)),
			newExpectingWriters("1", 1, "rendering call expression element 0: rendering literal int value int payload:"),
			newExpectingWriters("x", 1, "rendering call expression element 1: rendering ident Name:"),
			newExpectingWriters("20", 1, "rendering call expression element 2: rendering literal int value int payload:"),
			newExpectingWriters("y", 1, "rendering call expression Fn: rendering ident Name:"),
		},
		&syntax.Comprehension{
			Body: fooIdent,
			Clauses: []syntax.Node{
				&syntax.IfClause{Cond: fooCond},
				&syntax.ForClause{X: xIdent, Vars: yIdent},
			},
		}: {
			newExpectingWriters("[", 1, "rendering comprehension left token:"),
			newExpectingWriters("]", 1, "rendering comprehension right token:"),
			newExpectingWriters(" ", 6, "rendering comprehension space:"),
			newExpectingWriters("in", 1, "rendering comprehension IN token:"),
			newExpectingWriters("if", 1, "rendering comprehension IF token:"),
			newExpectingWriters("for", 1, "rendering comprehension FOR token:"),
			newExpectingWriters("y", 1, "rendering comprehension for clause Vars: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering comprehension for clause X: rendering ident Name:"),
			newExpectingWriters("foo_cond", 1, "rendering comprehension if clause Cond: rendering ident Name:"),
		},
		&syntax.Comprehension{
			Curly: true,
			Body:  fooIdent,
			Clauses: []syntax.Node{
				&syntax.IfClause{Cond: fooCond},
				&syntax.ForClause{X: xIdent, Vars: yIdent},
			},
		}: {
			newExpectingWriters("{", 1, "rendering comprehension left token:"),
			newExpectingWriters("}", 1, "rendering comprehension right token:"),
			newExpectingWriters(" ", 6, "rendering comprehension space:"),
			newExpectingWriters("in", 1, "rendering comprehension IN token:"),
			newExpectingWriters("if", 1, "rendering comprehension IF token:"),
			newExpectingWriters("for", 1, "rendering comprehension FOR token:"),
			newExpectingWriters("y", 1, "rendering comprehension for clause Vars: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering comprehension for clause X: rendering ident Name:"),
			newExpectingWriters("foo_cond", 1, "rendering comprehension if clause Cond: rendering ident Name:"),
		},
		&syntax.CondExpr{Cond: fooCond, True: xIdent, False: yIdent}: {
			newExpectingWriters("foo_cond", 1, "rendering condition expression Cond: rendering ident Name:"),
			newExpectingWriters("if", 1, "rendering condition expression IF token:"),
			newExpectingWriters("else", 1, "rendering condition expression ELSE token:"),
			newExpectingWriters(" ", 4, "rendering condition expression space:"),
			newExpectingWriters("y", 1, "rendering condition expression False: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering condition expression True: rendering ident Name:"),
		},
		&syntax.DictEntry{Key: xIdent, Value: yIdent}: {
			newExpectingWriters("y", 1, "rendering dict entry Value: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering dict entry Key: rendering ident Name:"),
			newExpectingWriters(":", 1, "rendering dict entry COLON token:"),
			newExpectingWriters(" ", 1, "rendering dict entry space:"),
		},
		&syntax.DictExpr{List: []syntax.Expr{&syntax.DictEntry{Key: xIdent, Value: yIdent}}}: {
			newExpectingWriters("{", 1, "rendering dict expression LBRACE token:"),
			newExpectingWriters("}", 1, "rendering dict expression RBRACE token:"),
			newExpectingWriters("y", 1, "rendering dict expression element 0: rendering dict entry Value: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering dict expression element 0: rendering dict entry Key: rendering ident Name:"),
			newExpectingWriters(":", 1, "rendering dict expression element 0: rendering dict entry COLON token:"),
			newExpectingWriters(" ", 1, "rendering dict expression element 0: rendering dict entry space:"),
			newExpectingWriters(",", 1, "rendering dict expression COMMA token:", WithDictOption(DictOptionMultilineComma)),
			newExpectingWriters("\n", 2, "rendering dict expression NEWLINE token:", WithDictOption(DictOptionMultilineComma)),
			newExpectingWriters("    ", 1, "rendering dict expression indent:", WithDictOption(DictOptionMultilineComma)),
		},
		&syntax.DotExpr{Name: yIdent, X: xIdent}: {
			newExpectingWriters("y", 1, "rendering dot expression Name: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering dot expression X: rendering ident Name:"),
			newExpectingWriters(".", 1, "rendering dot expression DOT token:"),
		},
		xIdent: {
			newExpectingWriters("x", 1, "rendering ident Name:"),
		},
		&syntax.IndexExpr{X: xIdent, Y: yIdent}: {
			newExpectingWriters("x", 1, "rendering index expression X: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering index expression Y: rendering ident Name:"),
			newExpectingWriters("[", 1, "rendering index expression LBRACK token:"),
			newExpectingWriters("]", 1, "rendering index expression RBRACK token:"),
		},
		&syntax.LambdaExpr{}: nil,
		&syntax.ListExpr{List: []syntax.Expr{
			fooIdent,
			xIdent,
			yIdent,
			twentyLiteral,
		}}: {
			newExpectingWriters("foo", 1, "rendering list expression element 0: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering list expression element 1: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering list expression element 2: rendering ident Name:"),
			newExpectingWriters("20", 1, "rendering list expression element 3: rendering literal int value int payload:"),
			newExpectingWriters("[", 1, "rendering list expression LBRACK token:"),
			newExpectingWriters("]", 1, "rendering list expression RBRACK token:"),
			newExpectingWriters(",", 3, "rendering list expression COMMA token:"),
			newExpectingWriters(" ", 3, "rendering list expression space:"),
			newExpectingWriters(",", 4, "rendering list expression COMMA token:", WithListOption(ListOptionMultilineComma)),
			newExpectingWriters("\n", 5, "rendering list expression NEWLINE token:", WithListOption(ListOptionMultilineComma)),
			newExpectingWriters("    ", 4, "rendering list expression indent:", WithListOption(ListOptionMultilineComma)),
		},
		twentyLiteral: {
			newExpectingWriters("20", 1, "rendering literal int value int payload:"),
		},
		&syntax.Literal{Value: "test"}: {
			newExpectingWriters(`"test"`, 1, "rendering literal string value string payload:"),
		},
		&syntax.Literal{Value: uint64(10)}: {
			newExpectingWriters("10", 1, "rendering literal uint64 value uint64 payload:"),
		},
		&syntax.Literal{Value: uint(10)}: {
			newExpectingWriters("10", 1, "rendering literal uint value uint payload:"),
		},
		&syntax.Literal{Value: int64(-10)}: {
			newExpectingWriters("-10", 1, "rendering literal int64 value int64 payload:"),
		},
		&syntax.Literal{Value: big.NewInt(-10)}: {
			newExpectingWriters("-10", 1, "rendering literal int64 value *big.Int payload:"),
		},
		&syntax.ParenExpr{X: xIdent}: {
			newExpectingWriters("x", 1, "rendering paren expression X: rendering ident Name:"),
			newExpectingWriters("(", 1, "rendering paren expression LPAREN token:"),
			newExpectingWriters(")", 1, "rendering paren expression RPAREN token:"),
		},
		&syntax.SliceExpr{Hi: fooIdent, X: xIdent, Lo: twentyLiteral, Step: oneLiteral}: {
			newExpectingWriters("[", 1, "rendering slice expression LBRACK token:"),
			newExpectingWriters("]", 1, "rendering slice expression RBRACK token:"),
			newExpectingWriters(":", 2, "rendering slice expression COLON token:"),
			newExpectingWriters("foo", 1, "rendering slice expression Hi: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering slice expression X: rendering ident Name:"),
			newExpectingWriters("1", 1, "rendering slice expression Step: rendering literal int value int payload:"),
			newExpectingWriters("20", 1, "rendering slice expression Lo: rendering literal int value int payload:"),
		},
		&syntax.TupleExpr{List: []syntax.Expr{xIdent, yIdent, twentyLiteral}}: {
			newExpectingWriters("x", 1, "rendering tuple expression element 0: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering tuple expression element 1: rendering ident Name:"),
			newExpectingWriters("20", 1, "rendering tuple expression element 2: rendering literal int value int payload:"),
			newExpectingWriters(",", 2, "rendering tuple expression COMMA token:"),
			newExpectingWriters(" ", 2, "rendering tuple expression space:"),
			newExpectingWriters(",", 3, "rendering tuple expression COMMA token:", WithTupleOption(TupleOptionMultilineMultipleComma)),
			newExpectingWriters("    ", 3, "rendering tuple expression indent:", WithTupleOption(TupleOptionMultilineMultipleComma)),
			newExpectingWriters("\n", 4, "rendering tuple expression NEWLINE token:", WithTupleOption(TupleOptionMultilineMultipleComma)),
		},
		&syntax.UnaryExpr{Op: syntax.MINUS, X: xIdent}: {
			newExpectingWriters("-", 1, "rendering unary expression, writing \"-\" token:"),
			newExpectingWriters("x", 1, "rendering unary expression X: rendering ident Name:"),
		},
		&syntax.UnaryExpr{Op: syntax.STAR}: {
			newExpectingWriters("*", 1, "rendering unary expression, writing \"*\" token:"),
		},

		// statements
		&syntax.AssignStmt{LHS: xIdent, Op: syntax.PLUS_EQ, RHS: yIdent}: {
			newExpectingWriters("x", 1, "rendering assignment statement LHS: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering assignment statement RHS: rendering ident Name:"),
			newExpectingWriters("+=", 1, "rendering assignment statement Op token:"),
			newExpectingWriters(" ", 2, "rendering assignment statement space:"),
			newExpectingWriters("\n", 1, "rendering assignment statement NEWLINE token:"),
		},
		&syntax.BranchStmt{Token: syntax.PASS}: {
			newExpectingWriters("pass", 1, "rendering branch statement Token token:"),
			newExpectingWriters("\n", 1, "rendering branch statement NEWLINE token:"),
		},
		&syntax.DefStmt{
			Name:   fooIdent,
			Params: []syntax.Expr{xIdent, yIdent},
			Body: []syntax.Stmt{
				&syntax.BranchStmt{Token: syntax.PASS},
			},
		}: {
			newExpectingWriters("foo", 1, "rendering def statement Name: rendering ident Name:"),
			newExpectingWriters("x", 1, "rendering def statement param 0: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering def statement param 1: rendering ident Name:"),
			newExpectingWriters("(", 1, "rendering def statement LPAREN token:"),
			newExpectingWriters(")", 1, "rendering def statement RPAREN token:"),
			newExpectingWriters(":", 1, "rendering def statement COLON token:"),
			newExpectingWriters("def", 1, "rendering def statement DEF token:"),
			newExpectingWriters(" ", 2, "rendering def statement space:"),
			newExpectingWriters(",", 1, "rendering def statement COMMA token:"),
			[]wantSetup{
				{
					writerSetup: newExpectingWriter("\n", 1, true),
					wantErr:     "rendering def statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 1",
				},
				{
					writerSetup: newExpectingWriter("\n", 2, true),
					wantErr:     "rendering def statement, rendering Body statement index 0: rendering branch statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 2",
				},
				{writerSetup: newExpectingWriter("\n", 3, false)},
			},
			newExpectingWriters("    ", 1, "rendering def statement, rendering Body statement index 0: rendering branch statement indent:"),
			newExpectingWriters("pass", 1, "rendering def statement, rendering Body statement index 0: rendering branch statement Token token:"),
		},
		&syntax.ExprStmt{X: xIdent}: {
			newExpectingWriters("x", 1, "rendering expression statement X: rendering ident Name:"),
		},
		&syntax.ForStmt{X: xIdent, Vars: yIdent, Body: []syntax.Stmt{
			&syntax.BranchStmt{Token: syntax.PASS},
		}}: {
			newExpectingWriters("x", 1, "rendering for statement X: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering for statement Vars: rendering ident Name:"),
			newExpectingWriters("for", 1, "rendering for statement FOR token:"),
			newExpectingWriters("in", 1, "rendering for statement IN token:"),
			newExpectingWriters(" ", 3, "rendering for statement space:"),
			newExpectingWriters(":", 1, "rendering for statement COLON token:"),
			newExpectingWriters("pass", 1, "rendering for statement, rendering Body statement index 0: rendering branch statement Token token:"),
			newExpectingWriters("    ", 1, "rendering for statement, rendering Body statement index 0: rendering branch statement indent:"),
			[]wantSetup{
				{
					writerSetup: newExpectingWriter("\n", 1, true),
					wantErr:     "rendering for statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 1",
				},
				{
					writerSetup: newExpectingWriter("\n", 2, true),
					wantErr:     "rendering for statement, rendering Body statement index 0: rendering branch statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 2",
				},
				{writerSetup: newExpectingWriter("\n", 3, false)},
			},
		},
		&syntax.IfStmt{
			Cond:  fooCond,
			False: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			True:  []syntax.Stmt{&syntax.BranchStmt{Token: syntax.BREAK}},
		}: {
			newExpectingWriters("foo_cond", 1, "rendering if statement Cond: rendering ident Name:"),
			newExpectingWriters("if", 1, "rendering if statement IF token:"),
			newExpectingWriters("else", 1, "rendering if statement ELSE token:"),
			newExpectingWriters("pass", 1, "rendering if statement, rendering False statement index 0: rendering branch statement Token token:"),
			newExpectingWriters("break", 1, "rendering if statement, rendering True statement index 0: rendering branch statement Token token:"),
			newExpectingWriters(" ", 1, "rendering if statement space:"),
			newExpectingWriters(":", 2, "rendering if statement COLON token:"),
			[]wantSetup{
				{
					writerSetup: newExpectingWriter("\n", 1, true),
					wantErr:     "rendering if statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 1",
				},
				{
					writerSetup: newExpectingWriter("\n", 2, true),
					wantErr:     "rendering if statement, rendering True statement index 0: rendering branch statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 2",
				},
				{
					writerSetup: newExpectingWriter("\n", 3, true),
					wantErr:     "rendering if statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 3",
				},
				{
					writerSetup: newExpectingWriter("\n", 4, true),
					wantErr:     "rendering if statement, rendering False statement index 0: rendering branch statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 4",
				},
				{writerSetup: newExpectingWriter("\n", 5, false)},
				{
					writerSetup: newExpectingWriter("    ", 1, true),
					wantErr:     "rendering if statement, rendering True statement index 0: rendering branch statement indent: AS EXPECTED: \"    \" occurence 1",
				},
				{
					writerSetup: newExpectingWriter("    ", 2, true),
					wantErr:     "rendering if statement, rendering False statement index 0: rendering branch statement indent: AS EXPECTED: \"    \" occurence 2",
				},
				{writerSetup: newExpectingWriter("    ", 3, false)},
			},
		},
		&syntax.LoadStmt{From: []*syntax.Ident{yIdent, fooIdent}, To: []*syntax.Ident{xIdent, {Name: "bar"}}, Module: &syntax.Literal{Value: "module"}}: {
			newExpectingWriters("x", 1, "rendering load statement To[0]: rendering ident Name:"),
			newExpectingWriters("bar", 1, "rendering load statement To[1]: rendering ident Name:"),
			newExpectingWriters("y", 1, "rendering load statement From[0]: rendering ident Name:"),
			newExpectingWriters("foo", 1, "rendering load statement From[1]: rendering ident Name:"),
			newExpectingWriters(`"module"`, 1, "rendering load statement Module: rendering literal string value string payload:"),
			newExpectingWriters("load", 1, "rendering load statement LOAD token:"),
			newExpectingWriters(" ", 2, "rendering load statement space:"),
			newExpectingWriters("=", 2, "rendering load statement EQ token:"),
			newExpectingWriters("(", 1, "rendering load statement LPAREN token:"),
			newExpectingWriters(")", 1, "rendering load statement RPAREN token:"),
			newExpectingWriters(",", 2, "rendering load statement COMMA token:"),
			newExpectingWriters("\"", 4, "rendering load statement quote:"),
			newExpectingWriters("\n", 1, "rendering load statement NEWLINE token:"),
		},
		&syntax.ReturnStmt{Result: xIdent}: {
			newExpectingWriters("x", 1, "rendering return statement Result: rendering ident Name:"),
			newExpectingWriters("return", 1, "rendering return statement RETURN token:"),
			newExpectingWriters(" ", 1, "rendering return statement space:"),
			newExpectingWriters("\n", 1, "rendering return statement NEWLINE token:"),
		},
		&syntax.WhileStmt{Cond: fooCond, Body: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}}}: {
			newExpectingWriters("while", 1, "rendering while statement WHILE token:"),
			newExpectingWriters(" ", 1, "rendering while statement space:"),
			newExpectingWriters("foo_cond", 1, "rendering while statement Cond: rendering ident Name:"),
			newExpectingWriters(":", 1, "rendering while statement COLON token:"),
			newExpectingWriters("    ", 1, "rendering while statement, rendering Body statement index 0: rendering branch statement indent:"),
			newExpectingWriters("pass", 1, "rendering while statement, rendering Body statement index 0: rendering branch statement Token token:"),
			[]wantSetup{
				{
					writerSetup: newExpectingWriter("\n", 1, true),
					wantErr:     "rendering while statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 1",
				},
				{
					writerSetup: newExpectingWriter("\n", 2, true),
					wantErr:     "rendering while statement, rendering Body statement index 0: rendering branch statement NEWLINE token: AS EXPECTED: \"\\n\" occurence 2",
				},
				{writerSetup: newExpectingWriter("\n", 3, false)},
			},
		},

		// breadcrumbs
		&syntax.ListExpr{List: []syntax.Expr{
			&syntax.BinaryExpr{
				Op: syntax.MINUS,
				X:  xIdent,
				Y:  &syntax.ListExpr{List: []syntax.Expr{twentyLiteral, fooIdent}},
			}}}: {
			[]wantSetup{{
				writerSetup: newExpectingWriter("foo", 1, true),
				wantErr:     "rendering list expression element 0: rendering binary expression Y: rendering list expression element 1: rendering ident Name: AS EXPECTED: \"foo\" occurence 1",
			}},
		},
	}
	for testNode, tests := range testMap {
		for _, subTests := range tests {
			for _, tt := range subTests {
				name := tt.wantErr
				if name == "" {
					name = fmt.Sprintf("%T", testNode)
				}
				ro, err := getOutputOpts(tt.opts...)
				if err != nil {
					t.Fatalf("invalid options: %v", err)
				}
				name = fmt.Sprintf("%s with %#v", name, ro)
				t.Run(name, func(t *testing.T) {
					var err error
					switch value := testNode.(type) {
					case syntax.Expr:
						err = WriteExpr(tt.writerSetup, value, tt.opts...)
					case syntax.Stmt:
						err = WriteStmt(tt.writerSetup, value, tt.opts...)
					default:
						t.Fatalf("unexpected type %T", value)
					}
					if tt.wantErr != "" {
						if err == nil {
							t.Fatalf("expected error %q, got nil", tt.wantErr)
						}
						if gotErr := err.Error(); tt.wantErr != "" && gotErr != tt.wantErr {
							t.Errorf("expected error %q, got %q", tt.wantErr, gotErr)
						}
						return
					}
					if err != nil {
						t.Fatalf("expected no error got %v", err)
					}
				})
			}
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

func Benchmark_huge_encapsulation_expr(b *testing.B) {
	const numRanges = 6

	ranges := make([]int, numRanges)
	ranges[0] = 1
	for i := 1; i < numRanges; i++ {
		ranges[i] = ranges[i-1] * 10
	}
	var (
		fooIdent   = &syntax.Ident{Name: "foo"}
		barIdent   = &syntax.Ident{Name: "bar"}
		oneLiteral = &syntax.Literal{Value: 1}
		tenLiteral = &syntax.Literal{Value: 10}
	)
	for _, num := range ranges {
		var x syntax.Expr = &syntax.ParenExpr{X: &syntax.Literal{Value: 1}}
		for i := 0; i < num; i++ {
			x = &syntax.IndexExpr{
				X: &syntax.DictExpr{List: []syntax.Expr{
					&syntax.DictEntry{Key: fooIdent, Value: oneLiteral},
					&syntax.DictEntry{Key: barIdent, Value: tenLiteral},
				}},
				Y: &syntax.ParenExpr{X: &syntax.UnaryExpr{Op: syntax.MINUS, X: x}},
			}
		}
		b.Run(strconv.Itoa(num), func(b *testing.B) {
			for tt := 0; tt < b.N; tt++ {
				err := WriteExpr(&nilWriter{}, x)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
