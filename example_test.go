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

func ExampleWithCallOption() {
	matrix := map[CallOption]string{
		CallOptionMultiline:                        "CallOptionMultiline",
		CallOptionMultilineComma:                   "CallOptionMultilineComma",
		CallOptionMultilineCommaTwoAndMore:         "CallOptionMultilineCommaTwoAndMore",
		CallOptionMultilineMultiple:                "CallOptionMultilineMultiple",
		CallOptionMultilineMultipleComma:           "CallOptionMultilineMultipleComma",
		CallOptionMultilineMultipleCommaTwoAndMore: "CallOptionMultilineMultipleCommaTwoAndMore",
		CallOptionSingleLine:                       "CallOptionSingleLine",
		CallOptionSingleLineComma:                  "CallOptionSingleLineComma",
		CallOptionSingleLineCommaTwoAndMore:        "CallOptionSingleLineCommaTwoAndMore",
	}
	for i := 0; i < len(matrix); i++ {
		opt, name := CallOption(i), matrix[CallOption(i)]
		call := &syntax.CallExpr{
			Fn: &syntax.Ident{Name: "some_func"},
			Args: []syntax.Expr{
				&syntax.Ident{Name: "foo"},
				&syntax.Ident{Name: "bar"},
			},
		}
		fmt.Println(name)
		for {
			exp, err := StarlarkExpr(call, WithCallOption(opt))

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(exp)

			if len(call.Args) == 0 {
				break
			}
			call.Args = call.Args[:len(call.Args)-1]
		}
	}
	// Output: CallOptionSingleLine
	// some_func(foo, bar)
	// some_func(foo)
	// some_func()
	// CallOptionSingleLineComma
	// some_func(foo, bar,)
	// some_func(foo,)
	// some_func()
	// CallOptionSingleLineCommaTwoAndMore
	// some_func(foo, bar,)
	// some_func(foo)
	// some_func()
	// CallOptionMultilineMultiple
	// some_func(
	//     foo,
	//     bar
	// )
	// some_func(foo)
	// some_func()
	// CallOptionMultilineMultipleComma
	// some_func(
	//     foo,
	//     bar,
	// )
	// some_func(foo,)
	// some_func()
	// CallOptionMultilineMultipleCommaTwoAndMore
	// some_func(
	//     foo,
	//     bar,
	// )
	// some_func(foo)
	// some_func()
	// CallOptionMultiline
	// some_func(
	//     foo,
	//     bar
	// )
	// some_func(
	//     foo
	// )
	// some_func()
	// CallOptionMultilineComma
	// some_func(
	//     foo,
	//     bar,
	// )
	// some_func(
	//     foo,
	// )
	// some_func()
	// CallOptionMultilineCommaTwoAndMore
	// some_func(
	//     foo,
	//     bar,
	// )
	// some_func(
	//     foo
	// )
	// some_func()
}

func ExampleWithDictOption() {
	matrix := map[DictOption]string{
		DictOptionMultiline:                        "DictOptionMultiline",
		DictOptionMultilineComma:                   "DictOptionMultilineComma",
		DictOptionMultilineCommaTwoAndMore:         "DictOptionMultilineCommaTwoAndMore",
		DictOptionMultilineMultiple:                "DictOptionMultilineMultiple",
		DictOptionMultilineMultipleComma:           "DictOptionMultilineMultipleComma",
		DictOptionMultilineMultipleCommaTwoAndMore: "DictOptionMultilineMultipleCommaTwoAndMore",
		DictOptionSingleLine:                       "DictOptionSingleLine",
		DictOptionSingleLineComma:                  "DictOptionSingleLineComma",
		DictOptionSingleLineCommaTwoAndMore:        "DictOptionSingleLineCommaTwoAndMore",
	}
	for i := 0; i < len(matrix); i++ {
		opt, name := DictOption(i), matrix[DictOption(i)]
		dict := &syntax.DictExpr{
			List: []syntax.Expr{
				&syntax.DictEntry{Key: &syntax.Ident{Name: "foo"}, Value: &syntax.Literal{Value: 1}},
				&syntax.DictEntry{Key: &syntax.Ident{Name: "bar"}, Value: &syntax.Literal{Value: 2}},
			},
		}
		fmt.Println(name)
		for {
			exp, err := StarlarkExpr(dict, WithDictOption(opt))

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(exp)

			if len(dict.List) == 0 {
				break
			}
			dict.List = dict.List[:len(dict.List)-1]
		}
	}
	// Output: DictOptionSingleLine
	// {foo: 1, bar: 2}
	// {foo: 1}
	// {}
	// DictOptionSingleLineComma
	// {foo: 1, bar: 2,}
	// {foo: 1,}
	// {}
	// DictOptionSingleLineCommaTwoAndMore
	// {foo: 1, bar: 2,}
	// {foo: 1}
	// {}
	// DictOptionMultilineMultiple
	// {
	//     foo: 1,
	//     bar: 2
	// }
	// {foo: 1}
	// {}
	// DictOptionMultilineMultipleComma
	// {
	//     foo: 1,
	//     bar: 2,
	// }
	// {foo: 1,}
	// {}
	// DictOptionMultilineMultipleCommaTwoAndMore
	// {
	//     foo: 1,
	//     bar: 2,
	// }
	// {foo: 1}
	// {}
	// DictOptionMultiline
	// {
	//     foo: 1,
	//     bar: 2
	// }
	// {
	//     foo: 1
	// }
	// {}
	// DictOptionMultilineComma
	// {
	//     foo: 1,
	//     bar: 2,
	// }
	// {
	//     foo: 1,
	// }
	// {}
	// DictOptionMultilineCommaTwoAndMore
	// {
	//     foo: 1,
	//     bar: 2,
	// }
	// {
	//     foo: 1
	// }
	// {}
}

func ExampleWithListOption() {
	matrix := map[ListOption]string{
		ListOptionMultiline:                        "ListOptionMultiline",
		ListOptionMultilineComma:                   "ListOptionMultilineComma",
		ListOptionMultilineCommaTwoAndMore:         "ListOptionMultilineCommaTwoAndMore",
		ListOptionMultilineMultiple:                "ListOptionMultilineMultiple",
		ListOptionMultilineMultipleComma:           "ListOptionMultilineMultipleComma",
		ListOptionMultilineMultipleCommaTwoAndMore: "ListOptionMultilineMultipleCommaTwoAndMore",
		ListOptionSingleLine:                       "ListOptionSingleLine",
		ListOptionSingleLineComma:                  "ListOptionSingleLineComma",
		ListOptionSingleLineCommaTwoAndMore:        "ListOptionSingleLineCommaTwoAndMore",
	}
	for i := 0; i < len(matrix); i++ {
		opt, name := ListOption(i), matrix[ListOption(i)]
		list := &syntax.ListExpr{
			List: []syntax.Expr{
				&syntax.Ident{Name: "foo"},
				&syntax.Ident{Name: "bar"},
			},
		}
		fmt.Println(name)
		for {
			exp, err := StarlarkExpr(list, WithListOption(opt))

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(exp)

			if len(list.List) == 0 {
				break
			}
			list.List = list.List[:len(list.List)-1]
		}
	}
	// Output: ListOptionSingleLine
	// [foo, bar]
	// [foo]
	// []
	// ListOptionSingleLineComma
	// [foo, bar,]
	// [foo,]
	// []
	// ListOptionSingleLineCommaTwoAndMore
	// [foo, bar,]
	// [foo]
	// []
	// ListOptionMultilineMultiple
	// [
	//     foo,
	//     bar
	// ]
	// [foo]
	// []
	// ListOptionMultilineMultipleComma
	// [
	//     foo,
	//     bar,
	// ]
	// [foo,]
	// []
	// ListOptionMultilineMultipleCommaTwoAndMore
	// [
	//     foo,
	//     bar,
	// ]
	// [foo]
	// []
	// ListOptionMultiline
	// [
	//     foo,
	//     bar
	// ]
	// [
	//     foo
	// ]
	// []
	// ListOptionMultilineComma
	// [
	//     foo,
	//     bar,
	// ]
	// [
	//     foo,
	// ]
	// []
	// ListOptionMultilineCommaTwoAndMore
	// [
	//     foo,
	//     bar,
	// ]
	// [
	//     foo
	// ]
	// []
}

func ExampleWithTupleOption() {
	matrix := map[TupleOption]string{
		TupleOptionMultiline:                        "TupleOptionMultiline",
		TupleOptionMultilineComma:                   "TupleOptionMultilineComma",
		TupleOptionMultilineCommaTwoAndMore:         "TupleOptionMultilineCommaTwoAndMore",
		TupleOptionMultilineMultiple:                "TupleOptionMultilineMultiple",
		TupleOptionMultilineMultipleComma:           "TupleOptionMultilineMultipleComma",
		TupleOptionMultilineMultipleCommaTwoAndMore: "TupleOptionMultilineMultipleCommaTwoAndMore",
		TupleOptionSingleLine:                       "TupleOptionSingleLine",
		TupleOptionSingleLineComma:                  "TupleOptionSingleLineComma",
		TupleOptionSingleLineCommaTwoAndMore:        "TupleOptionSingleLineCommaTwoAndMore",
	}
	for i := 0; i < len(matrix); i++ {
		opt, name := TupleOption(i), matrix[TupleOption(i)]

		tuple := &syntax.TupleExpr{
			List: []syntax.Expr{
				&syntax.Ident{Name: "foo"},
				&syntax.Ident{Name: "bar"},
			},
		}
		paren := &syntax.ParenExpr{X: tuple}

		fmt.Println(name)
		for {
			exp, err := StarlarkExpr(paren, WithTupleOption(opt))

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(exp)

			if len(tuple.List) == 0 {
				break
			}
			tuple.List = tuple.List[:len(tuple.List)-1]
		}
	}
	// Output: TupleOptionSingleLine
	// (foo, bar)
	// (foo)
	// ()
	// TupleOptionSingleLineComma
	// (foo, bar,)
	// (foo,)
	// ()
	// TupleOptionSingleLineCommaTwoAndMore
	// (foo, bar,)
	// (foo)
	// ()
	// TupleOptionMultilineMultiple
	// (
	//     foo,
	//     bar
	// )
	// (foo)
	// ()
	// TupleOptionMultilineMultipleComma
	// (
	//     foo,
	//     bar,
	// )
	// (foo,)
	// ()
	// TupleOptionMultilineMultipleCommaTwoAndMore
	// (
	//     foo,
	//     bar,
	// )
	// (foo)
	// ()
	// TupleOptionMultiline
	// (
	//     foo,
	//     bar
	// )
	// (
	//     foo
	// )
	// ()
	// TupleOptionMultilineComma
	// (
	//     foo,
	//     bar,
	// )
	// (
	//     foo,
	// )
	// ()
	// TupleOptionMultilineCommaTwoAndMore
	// (
	//     foo,
	//     bar,
	// )
	// (
	//     foo
	// )
	// ()
}
