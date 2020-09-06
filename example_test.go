package starlarkgen

import (
	"fmt"
	"log"
	"strings"

	"go.starlark.net/syntax"
)

func Example() {
	// Parse source file
	f, err := syntax.Parse("testdata/import.star", nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	// rename all the functions and function calls
	syntax.Walk(f, func(n syntax.Node) bool {
		switch t := n.(type) {
		case *syntax.DefStmt:
			t.Name.Name = "new_" + t.Name.Name
		case *syntax.CallExpr:
			if ident, ok := t.Fn.(*syntax.Ident); ok {
				ident.Name = "new_" + ident.Name
			}
		}
		return true
	})

	// Build the Starlark source back from the AST tree
	//
	// Note that node positions will be ignored
	var sb strings.Builder
	sep := ""
	for _, s := range f.Stmts {
		sb.WriteString(sep)
		st, err := StarlarkStmt(s)
		if err != nil {
			log.Fatal(err)
		}
		sb.WriteString(st)
		sep = "\n"
	}

	fmt.Println(sb.String())
	// Output: """test import file"""
	//
	// def new_foo(n):
	//     pass
	//
	// def new_bar(x):
	//     new_foo(x * 2)
}

func ExampleStarlarkStmt() {
	stm := &syntax.DefStmt{
		Name: &syntax.Ident{Name: "foo"},
		Params: []syntax.Expr{
			&syntax.Ident{Name: "bar"},
			&syntax.BinaryExpr{X: &syntax.Ident{Name: "foo"}, Op: syntax.EQ, Y: &syntax.Literal{Value: 2}},
		},
		Body: []syntax.Stmt{
			&syntax.ReturnStmt{Result: &syntax.BinaryExpr{X: &syntax.Ident{Name: "bar"}, Op: syntax.MINUS, Y: &syntax.Literal{Value: 2}}},
		},
	}

	st, err := StarlarkStmt(stm)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(st)
	// Output:def foo(bar, foo=2):
	//     return bar - 2
}

func ExampleWithSpaceEqBinary() {
	// foo=bar
	ex := &syntax.BinaryExpr{Op: syntax.EQ, X: &syntax.Ident{Name: "foo"}, Y: &syntax.Ident{Name: "bar"}}
	exWithSpace, err := StarlarkExpr(ex, WithSpaceEqBinary(true))
	if err != nil {
		log.Fatal(err)
	}
	exWithoutSpace, err := StarlarkExpr(ex, WithSpaceEqBinary(false)) // can be omitted, false is default
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(exWithSpace)
	fmt.Println(exWithoutSpace)
	// Output: foo = bar
	// foo=bar
}

func ExampleWithDepth() {
	st, err := StarlarkStmt(&syntax.BranchStmt{Token: syntax.PASS}, WithDepth(10))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(st) - len(strings.TrimLeft(st, " "))) // 10 * 4
	// Output: 40
}

func ExampleWithIndent() {
	st, err := StarlarkStmt(&syntax.IfStmt{
		Cond: &syntax.BinaryExpr{Op: syntax.LT, X: &syntax.Ident{Name: "foo"}, Y: &syntax.Ident{Name: "bar"}},
		True: []syntax.Stmt{&syntax.BranchStmt{Token: syntax.PASS}},
	}, WithIndent("____"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(st)
	// Output: if foo < bar:
	// ____pass
}
