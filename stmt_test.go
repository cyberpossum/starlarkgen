package starlarkgen

import (
	"strconv"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

func Test_stmt(t *testing.T) {
	tests := []struct {
		name string

		opts []Option

		inputAssignStmt *syntax.AssignStmt
		inputBranchStmt *syntax.BranchStmt
		inputDefStmt    *syntax.DefStmt
		inputExprStmt   *syntax.ExprStmt
		inputForStmt    *syntax.ForStmt
		inputIfStmt     *syntax.IfStmt
		inputLoadStmt   *syntax.LoadStmt
		inputReturnStmt *syntax.ReturnStmt
		inputWhileStmt  *syntax.WhileStmt

		want    string
		wantErr string
	}{
		{
			name:            "assign statement, EQ",
			inputAssignStmt: &syntax.AssignStmt{LHS: &syntax.Ident{Name: "foo"}, Op: syntax.EQ, RHS: &syntax.Literal{Value: 2}},
			want:            "foo = 2\n",
		},
		{
			name:            "assign statement, PLUS_EQ",
			inputAssignStmt: &syntax.AssignStmt{LHS: &syntax.Ident{Name: "foo"}, Op: syntax.PLUS_EQ, RHS: &syntax.Literal{Value: 2}},
			want:            "foo += 2\n",
		},
		{
			name:            "assign statement, invalid token",
			inputAssignStmt: &syntax.AssignStmt{LHS: &syntax.Ident{Name: "foo"}, Op: syntax.STAR, RHS: &syntax.Literal{Value: 2}},
			wantErr:         "rendering assign statement: unsupported Op token *, expected one of: =, +=, -=, *=, %=",
		},
		{
			name:            "branch statement, supported token PASS",
			inputBranchStmt: &syntax.BranchStmt{Token: syntax.PASS},
			want:            "pass\n",
		},
		{
			name:            "branch statement, supported token BREAK",
			inputBranchStmt: &syntax.BranchStmt{Token: syntax.BREAK},
			want:            "break\n",
		},
		{
			name:            "branch statement, supported token CONTINUE",
			inputBranchStmt: &syntax.BranchStmt{Token: syntax.CONTINUE},
			want:            "continue\n",
		},
		{
			name:            "branch statement, unsupported token",
			inputBranchStmt: &syntax.BranchStmt{Token: syntax.WHILE},
			wantErr:         "rendering branch statement: unsupported token while, expected break, continue or pass",
		},
		{
			name:         "def statement, no arguments",
			inputDefStmt: &syntax.DefStmt{Name: &syntax.Ident{Name: "foo"}, Body: []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Literal{Value: 0}}}},
			want:         "def foo():\n    return 0\n",
		},
		{
			name: "def statement, single argument",
			inputDefStmt: &syntax.DefStmt{
				Name:   &syntax.Ident{Name: "foo"},
				Params: []syntax.Expr{&syntax.Ident{Name: "bar"}},
				Body:   []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Ident{Name: "bar"}}},
			},
			want: "def foo(bar):\n    return bar\n",
		},
		{
			name: "def statement, args and kwargs",
			inputDefStmt: &syntax.DefStmt{
				Name: &syntax.Ident{Name: "foo"},
				Params: []syntax.Expr{
					&syntax.UnaryExpr{Op: syntax.STAR, X: &syntax.Ident{Name: "args"}},
					&syntax.UnaryExpr{Op: syntax.STARSTAR, X: &syntax.Ident{Name: "kwargs"}},
				},
				Body: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			},
			want: "def foo(*args, **kwargs):\n    pass\n",
		},
		{
			name: "def statement, multiple arguments with defaults",
			inputDefStmt: &syntax.DefStmt{
				Name: &syntax.Ident{Name: "foo"},
				Params: []syntax.Expr{
					&syntax.Ident{Name: "foo"},
					&syntax.Ident{Name: "bar"},
					&syntax.BinaryExpr{Op: syntax.EQ, X: &syntax.Ident{Name: "foobar"}, Y: &syntax.Literal{Value: 10}},
				},
				Body: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			},
			want: "def foo(foo, bar, foobar=10):\n    pass\n",
		},
		{
			name: "def statement, multiple arguments with defaults and spaces around EQ",
			inputDefStmt: &syntax.DefStmt{
				Name: &syntax.Ident{Name: "foo"},
				Params: []syntax.Expr{
					&syntax.Ident{Name: "foo"},
					&syntax.Ident{Name: "bar"},
					&syntax.BinaryExpr{Op: syntax.EQ, X: &syntax.Ident{Name: "foobar"}, Y: &syntax.Literal{Value: 10}},
				},
				Body: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			},
			opts: []Option{WithSpaceEqBinary(true)},
			want: "def foo(foo, bar, foobar = 10):\n    pass\n",
		},
		{
			name:          "expression statement, raw string",
			inputExprStmt: &syntax.ExprStmt{X: &syntax.Literal{Raw: "foo bar"}},
			want:          `foo bar` + "\n",
		},
		{
			name:          "expression statement, single-line docstring",
			inputExprStmt: &syntax.ExprStmt{X: &syntax.Literal{Value: "foo bar test"}},
			want:          `"""foo bar test"""` + "\n",
		},
		{
			name:          "expression statement, single-line docstring with triple quotes inside",
			inputExprStmt: &syntax.ExprStmt{X: &syntax.Literal{Value: "foo bar test \"\"\""}},
			want:          `"""foo bar test \"\"\""""` + "\n",
		},
		{
			name:          "expression statement, single-line docstring with triple quotes inside in the middle",
			inputExprStmt: &syntax.ExprStmt{X: &syntax.Literal{Value: "foo bar \"\"\" test"}},
			want:          `"""foo bar \"\"\" test"""` + "\n",
		},
		{
			name:          "expression statement, multi-line docstring",
			inputExprStmt: &syntax.ExprStmt{X: &syntax.Literal{Value: "foo bar test\ntest foo bar\ntest"}},
			want:          `    """foo bar test` + "\n" + `    test foo bar` + "\n" + `    test"""` + "\n",
			opts:          []Option{WithDepth(1)},
		},
		{
			name: "expression statement, multi-line docstring obtained from parser with empty lines",
			inputExprStmt: &syntax.ExprStmt{
				X: &syntax.Literal{
					Token:    syntax.STRING,
					TokenPos: syntax.Position{Col: 17, Line: 2},
					Value:    "some comment\n\n\n                more comment\n                even more comment\n                ",
				},
			},
			opts: []Option{WithDepth(1)},
			want: `    """some comment` + "\n\n\n" + `    more comment` + "\n" + `    even more comment` + "\n" + `    """` + "\n",
		},
		{
			name: "for statement",
			inputForStmt: &syntax.ForStmt{
				Vars: &syntax.Ident{Name: "x"},
				X: &syntax.ListExpr{List: []syntax.Expr{
					&syntax.Literal{Value: int64(1)},
					&syntax.Literal{Value: int64(2)},
					&syntax.Literal{Value: int64(3)},
				}},
				Body: []syntax.Stmt{
					&syntax.AssignStmt{LHS: &syntax.Ident{Name: "x"}, Op: syntax.PLUS_EQ, RHS: &syntax.Literal{Value: 1}},
					&syntax.ReturnStmt{Result: &syntax.Ident{Name: "x"}},
				},
			},
			want: "for x in [1, 2, 3]:\n    x += 1\n    return x\n",
		},
		{
			name: "if statement, no ELSE clause",
			inputIfStmt: &syntax.IfStmt{
				Cond: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.LT, Y: &syntax.Ident{Name: "bar"}},
				True: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			},
			want: "if foo < bar:\n    pass\n",
		},
		{
			name: "if statement, with ELSE clause",
			inputIfStmt: &syntax.IfStmt{
				Cond:  &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.LT, Y: &syntax.Ident{Name: "bar"}},
				True:  []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
				False: []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.STAR, Y: &syntax.Literal{Value: 2}}}},
			},
			want: "if foo < bar:\n    pass\nelse:\n    return foo * 2\n",
		},
		{
			// elif is desugared as an if-else chain by the parser
			name: "if statement, from parser ELIF chain statement",
			inputIfStmt: &syntax.IfStmt{
				Cond: &syntax.BinaryExpr{X: &syntax.Ident{Name: "a"}, Op: syntax.GT, Y: &syntax.Ident{Name: "b"}},
				True: []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Ident{Name: "a"}}},
				False: []syntax.Stmt{
					&syntax.IfStmt{
						Cond: &syntax.BinaryExpr{X: &syntax.Ident{Name: "b"}, Op: syntax.GT, Y: &syntax.Ident{Name: "c"}},
						True: []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Ident{Name: "b"}}},
						False: []syntax.Stmt{
							&syntax.IfStmt{
								Cond:  &syntax.BinaryExpr{X: &syntax.Ident{Name: "c"}, Op: syntax.GT, Y: &syntax.Ident{Name: "d"}},
								True:  []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Ident{Name: "c"}}},
								False: []syntax.Stmt{&syntax.ReturnStmt{Result: &syntax.Ident{Name: "d"}}},
							},
						},
					},
				},
			},
			want: "if a > b:\n    return a\nelse:\n    if b > c:\n        return b\n    else:\n        if c > d:\n            return c\n        else:\n            return d\n",
		},
		{
			name: "load, all named",
			inputLoadStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}},
				To:     []*syntax.Ident{{Name: "b"}, {Name: "a"}},
			},
			want: `load("foo.star", b="foo", a="bar")` + "\n",
		},
		{
			name: "load, none named",
			inputLoadStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}},
				To:     []*syntax.Ident{nil, nil},
			},
			want: `load("foo.star", "foo", "bar")` + "\n",
		},
		{
			name: "load, mixed",
			inputLoadStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}},
				To:     []*syntax.Ident{{Name: "foo"}, {Name: "a"}},
			},
			want: `load("foo.star", "foo", a="bar")` + "\n",
		},
		{
			name: "load, mixed, spaces around EQ",
			inputLoadStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}},
				To:     []*syntax.Ident{{Name: "foo"}, {Name: "a"}},
			},
			want: `load("foo.star", "foo", a = "bar")` + "\n",
			opts: []Option{WithSpaceEqBinary(true)},
		},
		{
			name: "load, lengths mismatch",
			inputLoadStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}, {Name: "foobar"}},
				To:     []*syntax.Ident{{Name: "b"}, {Name: "a"}},
			},
			wantErr: "rendering load statement, lengths mismatch, From: 3, To: 2",
		},
		{
			name:            "return without a value",
			inputReturnStmt: &syntax.ReturnStmt{},
			want:            "return\n",
		},
		{
			name: "return a single ident",
			inputReturnStmt: &syntax.ReturnStmt{
				Result: &syntax.Ident{Name: "n"},
			},
			want: "return n\n",
		},
		{
			name: "return expression",
			inputReturnStmt: &syntax.ReturnStmt{
				Result: &syntax.TupleExpr{List: []syntax.Expr{&syntax.Ident{Name: "i"}, &syntax.Ident{Name: "j"}}},
			},
			want: "return i, j\n",
		},
		{
			name: "while statement",
			inputWhileStmt: &syntax.WhileStmt{
				Cond: &syntax.BinaryExpr{X: &syntax.Ident{Name: "a"}, Op: syntax.GT, Y: &syntax.Ident{Name: "b"}},
				Body: []syntax.Stmt{&syntax.AssignStmt{LHS: &syntax.Ident{Name: "b"}, Op: syntax.PLUS_EQ, RHS: &syntax.Literal{Value: 1}}},
			},
			want: "while a > b:\n    b += 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				sb           strings.Builder
				exprSb       strings.Builder
				err          error
				errStmt      error
				inputStmt    syntax.Stmt
				opts, optErr = getOutputOpts(tt.opts...)
			)
			if optErr != nil {
				t.Fatalf("invalid options: %v", optErr)
			}

			switch {
			case tt.inputAssignStmt != nil:
				err = assignStmt(&sb, tt.inputAssignStmt, opts)
				inputStmt = tt.inputAssignStmt
			case tt.inputBranchStmt != nil:
				err = branchStmt(&sb, tt.inputBranchStmt, opts)
				inputStmt = tt.inputBranchStmt
			case tt.inputDefStmt != nil:
				err = defStmt(&sb, tt.inputDefStmt, opts)
				inputStmt = tt.inputDefStmt
			case tt.inputExprStmt != nil:
				err = exprStmt(&sb, tt.inputExprStmt, opts)
				inputStmt = tt.inputExprStmt
			case tt.inputForStmt != nil:
				err = forStmt(&sb, tt.inputForStmt, opts)
				inputStmt = tt.inputForStmt
			case tt.inputIfStmt != nil:
				err = ifStmt(&sb, tt.inputIfStmt, opts)
				inputStmt = tt.inputIfStmt
			case tt.inputLoadStmt != nil:
				err = loadStmt(&sb, tt.inputLoadStmt, opts)
				inputStmt = tt.inputLoadStmt
			case tt.inputReturnStmt != nil:
				err = returnStmt(&sb, tt.inputReturnStmt, opts)
				inputStmt = tt.inputReturnStmt
			case tt.inputWhileStmt != nil:
				err = whileStmt(&sb, tt.inputWhileStmt, opts)
				inputStmt = tt.inputWhileStmt
			default:
				t.Fatal("test value not set")
			}
			// check if stmt() provides same results as type-specific functions
			errStmt = stmt(&exprSb, inputStmt, opts)

			if tt.wantErr != "" {
				if err == nil || errStmt == nil {
					t.Fatal("expected an error, got nil")
				}
				if gotErr, gotStmtErr := err.Error(), errStmt.Error(); gotErr != tt.wantErr || gotStmtErr != tt.wantErr {
					t.Errorf("expected error %q, got %q and %q from stmt()", tt.wantErr, gotErr, gotStmtErr)
				}
				return
			}
			if err != nil || errStmt != nil {
				t.Fatalf("expected no error, got %v and %v from stmt()", err, errStmt)
			}
			if got, gotStmt := sb.String(), exprSb.String(); got != tt.want || gotStmt != tt.want {
				t.Errorf("expected %q, got %q and %q from stmt()", tt.want, got, gotStmt)
			}
		})
	}
}

func Test_nilStmt(t *testing.T) {
	tests := []struct {
		name         string
		input        syntax.Stmt
		wantErr      string
		wantNonEmpty bool
	}{
		{
			name:    "nil",
			wantErr: "unsupported type <nil>",
		},
		{
			name:    "nil syntax.AssignStmt",
			input:   (*syntax.AssignStmt)(nil),
			wantErr: "rendering assign statement: nil input",
		},
		{
			name:    "nil syntax.BranchStmt",
			input:   (*syntax.BranchStmt)(nil),
			wantErr: "rendering branch statement: nil input",
		},
		{
			name:    "nil syntax.DefStmt",
			input:   (*syntax.DefStmt)(nil),
			wantErr: "rendering def statement: nil input",
		},
		{
			name:    "nil syntax.ExprStmt",
			input:   (*syntax.ExprStmt)(nil),
			wantErr: "rendering expression statement: nil input",
		},
		{
			name:    "nil syntax.ForStmt",
			input:   (*syntax.ForStmt)(nil),
			wantErr: "rendering for statement: nil input",
		},
		{
			name:    "nil syntax.IfStmt",
			input:   (*syntax.IfStmt)(nil),
			wantErr: "rendering if statement: nil input",
		},
		{
			name:    "nil syntax.LoadStmt",
			input:   (*syntax.LoadStmt)(nil),
			wantErr: "rendering load statement: nil input",
		},
		{
			name:    "nil syntax.ReturnStmt",
			input:   (*syntax.ReturnStmt)(nil),
			wantErr: "rendering return statement: nil input",
		},
		{
			name:    "nil syntax.WhileStmt",
			input:   (*syntax.WhileStmt)(nil),
			wantErr: "rendering while statement: nil input",
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
			err := stmt(&sb, tt.input, defaultOpts.copy())

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

func Benchmark_huge_encapsulation_stmt(b *testing.B) {
	const numRanges = 5

	ranges := make([]int, numRanges)
	ranges[0] = 1
	for i := 1; i < numRanges; i++ {
		ranges[i] = ranges[i-1] * 10
	}
	var (
		fooIdent   = &syntax.Ident{Name: "foo"}
		barIdent   = &syntax.Ident{Name: "bar"}
		fooLessBar = &syntax.BinaryExpr{Op: syntax.LT, X: fooIdent, Y: barIdent}
		oneLiteral = &syntax.Literal{Value: 1}
		tenLiteral = &syntax.Literal{Value: 10}
	)
	for _, num := range ranges {
		var x syntax.Stmt = &syntax.BranchStmt{Token: syntax.PASS}
		for i := 0; i < num; i++ {
			x = &syntax.DefStmt{
				Name:   barIdent,
				Params: []syntax.Expr{fooIdent, barIdent},
				Body: []syntax.Stmt{
					&syntax.ExprStmt{X: oneLiteral},
					&syntax.ExprStmt{X: tenLiteral},
					&syntax.ExprStmt{X: &syntax.Literal{Value: "comment"}},
					&syntax.IfStmt{Cond: fooLessBar,
						True: []syntax.Stmt{
							&syntax.ForStmt{Vars: barIdent, X: fooIdent, Body: []syntax.Stmt{
								&syntax.ReturnStmt{Result: fooIdent},
							}},
						},
						False: []syntax.Stmt{
							&syntax.ReturnStmt{Result: barIdent},
						},
					},
					&syntax.WhileStmt{
						Cond: fooLessBar,
						Body: []syntax.Stmt{
							&syntax.LoadStmt{From: []*syntax.Ident{barIdent}, To: []*syntax.Ident{barIdent}, Module: &syntax.Literal{Value: "module"}},
							&syntax.AssignStmt{LHS: barIdent, Op: syntax.EQ, RHS: fooIdent},
							x,
							&syntax.ReturnStmt{Result: barIdent},
						},
					},
				},
			}
		}
		b.Run(strconv.Itoa(num), func(b *testing.B) {
			for tt := 0; tt < b.N; tt++ {
				err := WriteStmt(&nilWriter{}, x)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func Test_hasSpacePrefix(t *testing.T) {
	tests := []struct {
		name   string
		source string
		l      int
		want   bool
	}{
		{
			name:   "no spaces, 0 length",
			l:      0,
			source: "foo",
			want:   true,
		},
		{
			name:   "no spaces, >0 length",
			l:      2,
			source: "foo",
			want:   false,
		},
		{
			name:   "equal space count",
			l:      2,
			source: "  foo",
			want:   true,
		},
		{
			name:   "more spaces than expected",
			l:      2,
			source: "   foo",
			want:   true,
		},
		{
			name:   "less spaces than expected",
			l:      2,
			source: " foo",
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSpacePrefix([]byte(tt.source), tt.l); got != tt.want {
				t.Errorf("hasSpacePrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
