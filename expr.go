package starlarkgen

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func exprSequence(source []syntax.Expr, ro renderOption) []item {
	var (
		items        []item
		sep          []item
		prefixIndent bool
		lastComma    bool
		sourceLen    = len(source)
	)

	switch ro.multiLineType() {
	case multiLine:
		prefixIndent = sourceLen > 0
	case multiLineMultiple:
		prefixIndent = sourceLen > 1
	}
	switch ro.commaType() {
	case alwaysLastComma:
		lastComma = sourceLen > 0
	case lastCommaTwoAndMore:
		lastComma = sourceLen > 1
	}

	if prefixIndent {
		items = append(items,
			newlineItem,
			extraIndentItem,
		)
	}

	for i, arg := range source {
		items = append(items,
			sep...,
		)
		if prefixIndent {
			items = append(items,
				exprItemIndent(arg, fmt.Sprintf("element %d", i)),
			)
			sep = []item{tokenItem(syntax.COMMA, "COMMA"), newlineItem, extraIndentItem}
		} else {
			items = append(items,
				exprItem(arg, fmt.Sprintf("element %d", i)),
			)
			sep = commaSpace
		}
	}

	// add last comma if respective option is set
	if lastComma {
		items = append(items,
			tokenItem(syntax.COMMA, "COMMA"),
		)
	}
	// indent and newline for multiline
	if prefixIndent {
		items = append(items,
			newlineItem,
			indentItem,
		)
	}

	return items
}

func binaryExpr(out io.StringWriter, input *syntax.BinaryExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering binary expression: nil input")
	}

	items := []item{
		exprItem(input.X, "X"),
	}

	if input.Op != syntax.EQ || opts.spaceEqBinary {
		items = append(items,
			spaceItem,
		)
	}
	items = append(items, tokenItem(input.Op, "Op"))
	if input.Op != syntax.EQ || opts.spaceEqBinary {
		items = append(items,
			spaceItem,
		)
	}

	items = append(items,
		exprItem(input.Y, "Y"),
	)

	return render(out, "rendering binary expression", opts, items...)
}

func callExpr(out io.StringWriter, input *syntax.CallExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering call expression: nil input")
	}

	items := []item{
		exprItem(input.Fn, "Fn"),
		tokenItem(syntax.LPAREN, "LPAREN"),
	}

	items = append(items, exprSequence(input.Args, renderOption(opts.callOption))...)

	items = append(items,
		tokenItem(syntax.RPAREN, "RPAREN"),
	)

	return render(out, "rendering call expression", opts, items...)
}

func comprehension(out io.StringWriter, input *syntax.Comprehension, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering comprehension: nil input")
	}

	tokens := []syntax.Token{syntax.LBRACK, syntax.RBRACK}
	// when input.Curly is true, use {...} instead
	if input.Curly {
		tokens = []syntax.Token{syntax.LBRACE, syntax.RBRACE}
	}

	items := []item{
		tokenItem(tokens[0], "left token"),
		exprItem(input.Body, "Body"),
	}

	for _, cl := range input.Clauses {
		switch t := cl.(type) {
		case *syntax.ForClause:
			items = append(items,
				spaceItem,
				tokenItem(syntax.FOR, "FOR"),
				spaceItem,
				exprItem(t.Vars, "for clause Vars"),
				spaceItem,
				tokenItem(syntax.IN, "IN"),
				spaceItem,
				exprItem(t.X, "for clause X"),
			)
		case *syntax.IfClause:
			items = append(items,
				spaceItem,
				tokenItem(syntax.IF, "IF"),
				spaceItem,
				exprItem(t.Cond, "if clause Cond"),
			)
		default:
			return fmt.Errorf("unexpected clause type %T rendering comprehension", t)
		}
	}

	items = append(items,
		tokenItem(tokens[1], "right token"),
	)

	return render(out, "rendering comprehension", opts, items...)
}

func condExpr(out io.StringWriter, input *syntax.CondExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering condition expression: nil input")
	}

	return render(out, "rendering condition expression", opts,
		exprItem(input.True, "True"),
		spaceItem,
		tokenItem(syntax.IF, "IF"),
		spaceItem,
		exprItem(input.Cond, "Cond"),
		spaceItem,
		tokenItem(syntax.ELSE, "ELSE"),
		spaceItem,
		exprItem(input.False, "False"),
	)
}

func dictEntry(out io.StringWriter, input *syntax.DictEntry, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering dict entry: nil input")
	}
	return render(out, "rendering dict entry", opts,
		exprItem(input.Key, "Key"),
		colonItem,
		spaceItem,
		exprItem(input.Value, "Value"),
	)
}

func dictExpr(out io.StringWriter, input *syntax.DictExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering dict expression: nil input")
	}

	// validate the dict elements
	for _, elem := range input.List {
		if _, ok := elem.(*syntax.DictEntry); !ok {
			return fmt.Errorf("expected *syntax.DictEntry, got %T in dictExpr", elem)
		}
	}

	items := []item{
		tokenItem(syntax.LBRACE, "LBRACE"),
	}

	items = append(items, exprSequence(input.List, renderOption(opts.dictOption))...)

	items = append(items, tokenItem(syntax.RBRACE, "RBRACE"))

	return render(out, "rendering dict expression", opts, items...)
}

func dotExpr(out io.StringWriter, input *syntax.DotExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering dot expression: nil input")
	}

	return render(out, "rendering dot expression", opts,
		exprItem(input.X, "X"),
		tokenItem(syntax.DOT, "DOT"),
		exprItem(input.Name, "Name"),
	)
}

func ident(out io.StringWriter, input *syntax.Ident, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering ident: nil input")
	}

	return render(out, "rendering ident", opts, stringItem(input.Name, "Name"))
}

func indexExpr(out io.StringWriter, input *syntax.IndexExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering index expression: nil input")
	}

	return render(out, "rendering index expression", opts,
		exprItem(input.X, "X"),
		tokenItem(syntax.LBRACK, "LBRACK"),
		exprItem(input.Y, "Y"),
		tokenItem(syntax.RBRACK, "RBRACK"),
	)
}

func listExpr(out io.StringWriter, input *syntax.ListExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering list expression: nil input")
	}

	items := []item{
		tokenItem(syntax.LBRACK, "LBRACK"),
	}

	items = append(items, exprSequence(input.List, renderOption(opts.listOption))...)

	items = append(items, tokenItem(syntax.RBRACK, "RBRACK"))
	return render(out, "rendering list expression", opts, items...)
}

func literal(out io.StringWriter, input *syntax.Literal, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering literal: nil input")
	}

	if input.Value == nil {
		return render(out, "rendering literal raw", opts, stringItem(input.Raw, "raw value"))
	}

	switch t := input.Value.(type) {
	case string:
		return render(out, "rendering literal string value", opts, stringItem(starlark.String(t).String(), "string payload"))
	case int:
		return render(out, "rendering literal int value", opts, stringItem(starlark.MakeInt(t).String(), "int payload"))
	case uint:
		return render(out, "rendering literal uint value", opts, stringItem(starlark.MakeUint(t).String(), "uint payload"))
	case int64:
		return render(out, "rendering literal int64 value", opts, stringItem(starlark.MakeInt64(t).String(), "int64 payload"))
	case uint64:
		return render(out, "rendering literal uint64 value", opts, stringItem(starlark.MakeUint64(t).String(), "uint64 payload"))
	case *big.Int:
		if t == nil {
			return errors.New("nil literal *big.Int value provided")
		}
		return render(out, "rendering literal int64 value", opts, stringItem(starlark.MakeBigInt(t).String(), "*big.Int payload"))
	default:
		return fmt.Errorf("unsupported literal value type %T, expected string, int, int64, uint, uint64 or *big.Int", t)
	}
}

func parenExpr(out io.StringWriter, input *syntax.ParenExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering paren expression: nil input")
	}

	return render(out, "rendering paren expression", opts,
		tokenItem(syntax.LPAREN, "LPAREN"),
		exprItem(input.X, "X"),
		tokenItem(syntax.RPAREN, "RPAREN"),
	)
}

func sliceExpr(out io.StringWriter, input *syntax.SliceExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering slice expression: nil input")
	}

	items := []item{
		exprItem(input.X, "X"),
		tokenItem(syntax.LBRACK, "LBRACK"),
	}

	if input.Lo != nil {
		items = append(items,
			exprItem(input.Lo, "Lo"),
		)
	}

	items = append(items,
		colonItem,
	)

	if input.Hi != nil {
		items = append(items,
			exprItem(input.Hi, "Hi"),
		)
	}
	if input.Step != nil {
		items = append(items,
			colonItem,
			exprItem(input.Step, "Step"),
		)
	}
	items = append(items,
		tokenItem(syntax.RBRACK, "RBRACK"),
	)
	return render(out, "rendering slice expression", opts, items...)
}

func tupleExpr(out io.StringWriter, input *syntax.TupleExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering tuple expression: nil input")
	}

	return render(out, "rendering tuple expression", opts, exprSequence(input.List, renderOption(opts.tupleOption))...)
}

func unaryExpr(out io.StringWriter, input *syntax.UnaryExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering unary expression: nil input")
	}

	// from the go.starlark.net docs:
	//
	// As a special case, UnaryOp{Op:Star} may also represent
	// the star parameter in def f(*args) or def f(*, x).
	if input.Op == syntax.STAR && input.X == nil {
		return render(out, "rendering unary expression", opts,
			tokenItem(syntax.STAR, "STAR"),
		)
	}

	return render(out, "rendering unary expression", opts,
		tokenItem(input.Op, "Op"),
		exprItem(input.X, "X"),
	)
}

func expr(out io.StringWriter, input syntax.Expr, opts *outputOpts) error {
	switch t := input.(type) {
	case *syntax.BinaryExpr:
		return binaryExpr(out, t, opts)
	case *syntax.CallExpr:
		return callExpr(out, t, opts)
	case *syntax.Comprehension:
		return comprehension(out, t, opts)
	case *syntax.CondExpr:
		return condExpr(out, t, opts)
	case *syntax.DictEntry:
		return dictEntry(out, t, opts)
	case *syntax.DictExpr:
		return dictExpr(out, t, opts)
	case *syntax.DotExpr:
		return dotExpr(out, t, opts)
	case *syntax.Ident:
		return ident(out, t, opts)
	case *syntax.IndexExpr:
		return indexExpr(out, t, opts)
	case *syntax.ListExpr:
		return listExpr(out, t, opts)
	case *syntax.Literal:
		return literal(out, t, opts)
	case *syntax.ParenExpr:
		return parenExpr(out, t, opts)
	case *syntax.SliceExpr:
		return sliceExpr(out, t, opts)
	case *syntax.TupleExpr:
		return tupleExpr(out, t, opts)
	case *syntax.UnaryExpr:
		return unaryExpr(out, t, opts)
	default:
		// e.g. *syntax.LambdaExpr
		return fmt.Errorf("type %T is not supported", t)
	}
}
