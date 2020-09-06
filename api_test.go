package starlarkgen

import (
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

func Test_getOutputOpts(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    *outputOpts
		wantErr string
	}{
		{
			name: "defaults",
			want: &defaultOpts,
		},
		{
			name:    "custom indent",
			options: []Option{WithIndent("\t")},
			want: &outputOpts{
				depth:         defaultDepth,
				indent:        "\t",
				spaceEqBinary: defaultSpaceEqBinary,
			},
		},
		{
			name:    "custom depth",
			options: []Option{WithDepth(10)},
			want: &outputOpts{
				depth:         10,
				indent:        defaultIndent,
				spaceEqBinary: defaultSpaceEqBinary,
			},
		},
		{
			name:    "invalid depth",
			options: []Option{WithDepth(-1)},
			wantErr: "invalid depth value -1, value must be >= 0",
		},
		{
			name:    "without spaces around binary",
			options: []Option{WithSpaceEqBinary(false)},
			want: &outputOpts{
				depth:         defaultDepth,
				indent:        defaultIndent,
				spaceEqBinary: false,
			},
		},
		{
			name:    "with spaces around binary",
			options: []Option{WithSpaceEqBinary(true)},
			want: &outputOpts{
				depth:         defaultDepth,
				indent:        defaultIndent,
				spaceEqBinary: true,
			},
		},
		{
			name:    "multiple options chained",
			options: []Option{WithSpaceEqBinary(true), WithDepth(10), WithIndent("\t")},
			want: &outputOpts{
				depth:         10,
				indent:        "\t",
				spaceEqBinary: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getOutputOpts(tt.options...)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got != nil {
					t.Fatalf("expected nil result on error, got %v", got)
				}
				if gotErr := err.Error(); gotErr != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, gotErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOutputOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStarlark(t *testing.T) {
	tests := []struct {
		name      string
		inputStmt syntax.Stmt
		inputExpr syntax.Expr
		options   []Option
		want      string
		wantErr   string
	}{
		{
			name: "statement, success",
			inputStmt: &syntax.LoadStmt{
				Module: &syntax.Literal{Value: "foo.star"},
				From:   []*syntax.Ident{{Name: "foo"}, {Name: "bar"}},
				To:     []*syntax.Ident{{Name: "foo"}, {Name: "a"}},
			},
			want: "load(\"foo.star\", \"foo\", a=\"bar\")\n",
		},
		{
			name:      "statement failure, invalid options",
			inputStmt: &syntax.BranchStmt{Token: syntax.PASS},
			options:   []Option{WithDepth(-1)},
			wantErr:   "invalid depth value -1, value must be >= 0",
		},
		{
			name:      "statement failure, invalid input",
			inputStmt: &syntax.BranchStmt{Token: syntax.ILLEGAL},
			wantErr:   "rendering branch statement: unsupported token illegal token, expected break, continue or pass",
		},
		{
			name:      "expression, success",
			inputExpr: &syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.LT, Y: &syntax.Ident{Name: "bar"}},
			want:      "foo < bar",
		},
		{
			name:      "expression failure, invalid options",
			inputExpr: &syntax.LambdaExpr{},
			options:   []Option{WithDepth(-1)},
			wantErr:   "invalid depth value -1, value must be >= 0",
		},
		{
			name:      "expression failure, invalid input",
			inputExpr: &syntax.LambdaExpr{},
			wantErr:   "type *syntax.LambdaExpr is not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got, gotW string
				err, errW error
				sb        strings.Builder
			)
			if tt.inputExpr != nil {
				got, err = StarlarkExpr(tt.inputExpr, tt.options...)
				errW = WriteExpr(&sb, tt.inputExpr, tt.options...)
			}
			if tt.inputStmt != nil {
				got, err = StarlarkStmt(tt.inputStmt, tt.options...)
				errW = WriteStmt(&sb, tt.inputStmt, tt.options...)
			}
			gotW = sb.String()
			if tt.wantErr != "" {
				if err == nil || errW == nil {
					t.Fatal("expected error, got nil")
				}
				if got != "" || gotW != "" {
					t.Fatalf("expected empty result on error, got %q and %q from Write...", got, gotW)
				}
				if gotErr, gotWErr := err.Error(), errW.Error(); gotErr != tt.wantErr || gotWErr != tt.wantErr {
					t.Fatalf("expected error %q, got %q and %q from Write...", tt.wantErr, gotErr, gotWErr)
				}
				return
			}
			if err != nil || errW != nil {
				t.Fatalf("expected no error, got %v and %v from Write...", err, errW)
			}
			if got != tt.want || gotW != tt.want {
				t.Errorf("want %q, got %q and %q from Write...", tt.want, got, gotW)
			}
		})
	}
}
