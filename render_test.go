package starlarkgen

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

type writeStringFunc func(string) (int, error)

func (f writeStringFunc) WriteString(s string) (int, error) {
	return f(s)
}

func Test_exprItem(t *testing.T) {
	want := &item{
		itemType:  exprType,
		expr:      &syntax.Ident{Name: "foo"},
		valueDesc: "test desc",
	}

	if got := exprItem(&syntax.Ident{Name: "foo"}, "test desc"); !reflect.DeepEqual(got, want) {
		t.Errorf("exprItem() = %v, want %v", got, want)
	}
}

func Test_stringItem(t *testing.T) {
	want := &item{
		itemType:  stringType,
		value:     "foo",
		valueDesc: "test desc",
	}

	if got := stringItem("foo", "test desc"); !reflect.DeepEqual(got, want) {
		t.Errorf("exprItem() = %v, want %v", got, want)
	}
}

func Test_tokenItem(t *testing.T) {
	want := &item{
		itemType:  tokenType,
		token:     syntax.PASS,
		valueDesc: "test desc",
	}

	if got := tokenItem(syntax.PASS, "test desc"); !reflect.DeepEqual(got, want) {
		t.Errorf("exprItem() = %v, want %v", got, want)
	}
}

func Test_stmtsItem(t *testing.T) {
	tests := []struct {
		name      string
		stmts     []syntax.Stmt
		addIndent bool
		want      *item
	}{
		{
			name:  "no indent",
			stmts: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			want: &item{
				itemType:  stmtsType,
				addIndent: 0,
				stmts:     []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
				valueDesc: "no indent",
			},
		},
		{
			name:      "with indent",
			stmts:     []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
			addIndent: true,
			want: &item{
				itemType:  stmtsType,
				addIndent: 1,
				stmts:     []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
				valueDesc: "with indent",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stmtsItem(tt.stmts, tt.name, tt.addIndent); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stmtsItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_render(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name   string
		opts   []Option
		items  []*item
		writer writeStringFunc

		wantErr string
		want    string
	}{
		{
			name: "nil",
		},
		{
			name:    "unsupported type",
			items:   []*item{{}},
			wantErr: "test unsupported type: item type 0 is not supported in render",
		},
		{
			name:    "nil element in sequence",
			items:   []*item{indentItem, spaceItem, nil, spaceItem, indentItem},
			wantErr: "nil item in render, errPrefix: test nil element in sequence",
		},
		{
			name:  "indent at zero level",
			items: []*item{indentItem},
		},
		{
			name:  "indent",
			items: []*item{indentItem},
			want:  "    ",
			opts:  []Option{WithDepth(1)},
		},
		{
			name:  "indent, custom",
			items: []*item{indentItem},
			want:  "\t\t",
			opts:  []Option{WithDepth(2), WithIndent("\t")},
		},
		{
			name:  "error writing indent",
			items: []*item{indentItem},
			opts:  []Option{WithDepth(1)},
			writer: func(string) (int, error) {
				return 0, testError
			},
			wantErr: "test error writing indent indent: test error",
		},
		{
			name:  "expression",
			items: []*item{exprItem(&syntax.Ident{Name: "foo"}, "FOO")},
			want:  "foo",
		},
		{
			name:    "expression with rendering error",
			items:   []*item{exprItem(&syntax.LambdaExpr{}, "FOO")},
			wantErr: "test expression with rendering error FOO: type *syntax.LambdaExpr is not supported",
		},
		{
			name: "statements, no added indent",
			items: []*item{stmtsItem([]syntax.Stmt{
				&syntax.BranchStmt{Token: syntax.PASS},
				&syntax.BranchStmt{Token: syntax.PASS},
			}, "STMTS", false)},
			want: "pass\npass\n",
		},
		{
			name: "statements, added indent",
			items: []*item{stmtsItem([]syntax.Stmt{
				&syntax.BranchStmt{Token: syntax.PASS},
				&syntax.BranchStmt{Token: syntax.PASS},
			}, "STMTS", true)},
			want: "    pass\n    pass\n",
		},
		{
			name: "statements, failure",
			items: []*item{stmtsItem([]syntax.Stmt{
				&syntax.BranchStmt{Token: syntax.PASS},
				&syntax.BranchStmt{Token: syntax.PASS},
				&syntax.ExprStmt{X: &syntax.LambdaExpr{}},
			}, "STMTS", true)},
			wantErr: "test statements, failure, rendering STMTS statement index 2: rendering expression statement X: type *syntax.LambdaExpr is not supported",
		},
		{
			name:  "string",
			items: []*item{stringItem("foo", "FOO")},
			want:  "foo",
		},
		{
			name:  "string failure",
			items: []*item{stringItem("foo", "FOO")},
			writer: func(string) (int, error) {
				return 0, testError
			},
			wantErr: "test string failure FOO: test error",
		},
		{
			name:  "newline",
			items: []*item{newlineItem},
			want:  "\n",
		},
		{
			name:  "newline failure",
			items: []*item{newlineItem},
			writer: func(string) (int, error) {
				return 0, testError
			},
			wantErr: "test newline failure NEWLINE token: test error",
		},
		{
			name:  "token",
			items: []*item{tokenItem(syntax.EQL, "EQL")},
			want:  "==",
		},
		{
			name:    "illegal token",
			items:   []*item{tokenItem(syntax.ILLEGAL, "ILLEGAL")},
			wantErr: "test illegal token ILLEGAL token: illegal token not supported",
		},
		{
			name:  "token failure",
			items: []*item{tokenItem(syntax.EQL, "EQL")},
			writer: func(string) (int, error) {
				return 0, testError
			},
			wantErr: "test token failure EQL token: test error",
		},
		{
			name:  "sequence",
			items: []*item{quoteItem, spaceItem, tokenItem(syntax.EQL, "EQL"), spaceItem, quoteItem},
			want:  `" == "`,
		},
		{
			name:    "sequence failure",
			items:   []*item{quoteItem, spaceItem, tokenItem(syntax.EQL, "EQL"), spaceItem, quoteItem, tokenItem(syntax.ILLEGAL, "ILLEGAL")},
			wantErr: "test sequence failure ILLEGAL token: illegal token not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				sb strings.Builder
				w  io.StringWriter = &sb
			)
			opts, err := getOutputOpts(tt.opts...)
			if err != nil {
				t.Fatalf("error parsing options: %v", err)
			}
			if tt.writer != nil {
				w = tt.writer
			}
			err = render(w, "test "+tt.name, opts, tt.items...)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if gotErr := err.Error(); gotErr != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, gotErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got := sb.String(); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
