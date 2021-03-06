package starlarkgen

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"unsafe"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const (
	newline = "\n"
	space   = " "
	quote   = "\""
)

type sepType uint8

const (
	sepNone sepType = iota
	sepCommaSpace
	sepCommaNewlineIndent
)

func writeRepeat(out io.StringWriter, source string, n int) error {
	for i := 0; i < n; i++ {
		if _, err := out.WriteString(source); err != nil {
			return err
		}
	}

	return nil
}

func exprSequence(out io.StringWriter, source []syntax.Expr, ro renderOption, opts *outputOpts) error {
	var (
		sep          sepType
		prefixIndent bool
		lastComma    bool
		sourceLen    = len(source)
		expOpts      *outputOpts
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
		expOpts = opts.addDepth(1)
		if _, err := out.WriteString(newline); err != nil {
			return fmt.Errorf("NEWLINE token: %w", err)
		}
		if err := writeRepeat(out, expOpts.indent, expOpts.depth); err != nil {
			return fmt.Errorf("indent: %w", err)
		}
	}

	for i, arg := range source {
		switch sep {
		case sepNone:
		case sepCommaSpace:
			if _, err := out.WriteString(syntax.COMMA.String()); err != nil {
				return fmt.Errorf("COMMA token: %w", err)
			}
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("space: %w", err)
			}
		case sepCommaNewlineIndent:
			if _, err := out.WriteString(syntax.COMMA.String()); err != nil {
				return fmt.Errorf("COMMA token: %w", err)
			}
			if _, err := out.WriteString(newline); err != nil {
				return fmt.Errorf("NEWLINE token: %w", err)
			}
			if err := writeRepeat(out, expOpts.indent, expOpts.depth); err != nil {
				return fmt.Errorf("indent: %w", err)
			}
		}
		if prefixIndent {
			if err := expr(out, arg, expOpts); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
			sep = sepCommaNewlineIndent
		} else {
			if err := expr(out, arg, opts); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
			sep = sepCommaSpace
		}
	}

	// add last comma if respective option is set
	if lastComma {
		if _, err := out.WriteString(syntax.COMMA.String()); err != nil {
			return fmt.Errorf("COMMA token: %w", err)
		}
	}
	// indent and newline for multiline
	if prefixIndent {
		if _, err := out.WriteString(newline); err != nil {
			return fmt.Errorf("NEWLINE token: %w", err)
		}
		if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
			return fmt.Errorf("indent: %w", err)
		}
	}

	return nil
}

func binaryExpr(out io.StringWriter, input *syntax.BinaryExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering binary expression: nil input")
	}

	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering binary expression X: %w", err)
	}

	if input.Op != syntax.EQ || opts.spaceEqBinary {
		if _, err := out.WriteString(space); err != nil {
			return fmt.Errorf("rendering binary expression space: %w", err)
		}
	}

	if _, err := out.WriteString(input.Op.String()); err != nil {
		return fmt.Errorf("rendering binary expression Op token: %w", err)
	}

	if input.Op != syntax.EQ || opts.spaceEqBinary {
		if _, err := out.WriteString(space); err != nil {
			return fmt.Errorf("rendering binary expression space: %w", err)
		}
	}

	if err := expr(out, input.Y, opts); err != nil {
		return fmt.Errorf("rendering binary expression Y: %w", err)
	}

	return nil
}

func callExpr(out io.StringWriter, input *syntax.CallExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering call expression: nil input")
	}

	if err := expr(out, input.Fn, opts); err != nil {
		return fmt.Errorf("rendering call expression Fn: %w", err)
	}

	if _, err := out.WriteString(syntax.LPAREN.String()); err != nil {
		return fmt.Errorf("rendering call expression LPAREN token: %w", err)
	}

	if err := exprSequence(out, input.Args, renderOption(opts.callOption), opts); err != nil {
		return fmt.Errorf("rendering call expression: %w", err)
	}

	if _, err := out.WriteString(syntax.RPAREN.String()); err != nil {
		return fmt.Errorf("rendering call expression RPAREN token: %w", err)
	}

	return nil
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

	if _, err := out.WriteString(tokens[0].String()); err != nil {
		return fmt.Errorf("rendering comprehension left token: %w", err)
	}

	if err := expr(out, input.Body, opts); err != nil {
		return fmt.Errorf("rendering comprehension Body: %w", err)
	}

	for _, cl := range input.Clauses {
		switch t := cl.(type) {
		case *syntax.ForClause:
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if _, err := out.WriteString(syntax.FOR.String()); err != nil {
				return fmt.Errorf("rendering comprehension FOR token: %w", err)
			}
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if err := expr(out, t.Vars, opts); err != nil {
				return fmt.Errorf("rendering comprehension for clause Vars: %w", err)
			}
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if _, err := out.WriteString(syntax.IN.String()); err != nil {
				return fmt.Errorf("rendering comprehension IN token: %w", err)
			}
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if err := expr(out, t.X, opts); err != nil {
				return fmt.Errorf("rendering comprehension for clause X: %w", err)
			}
		case *syntax.IfClause:
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if _, err := out.WriteString(syntax.IF.String()); err != nil {
				return fmt.Errorf("rendering comprehension IF token: %w", err)
			}
			if _, err := out.WriteString(space); err != nil {
				return fmt.Errorf("rendering comprehension space: %w", err)
			}
			if err := expr(out, t.Cond, opts); err != nil {
				return fmt.Errorf("rendering comprehension if clause Cond: %w", err)
			}
		default:
			return fmt.Errorf("unexpected clause type %T rendering comprehension", t)
		}
	}

	if _, err := out.WriteString(tokens[1].String()); err != nil {
		return fmt.Errorf("rendering comprehension right token: %w", err)
	}

	return nil
}

func condExpr(out io.StringWriter, input *syntax.CondExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering condition expression: nil input")
	}

	if err := expr(out, input.True, opts); err != nil {
		return fmt.Errorf("rendering condition expression True: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering condition expression space: %w", err)
	}
	if _, err := out.WriteString(syntax.IF.String()); err != nil {
		return fmt.Errorf("rendering condition expression IF token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering condition expression space: %w", err)
	}
	if err := expr(out, input.Cond, opts); err != nil {
		return fmt.Errorf("rendering condition expression Cond: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering condition expression space: %w", err)
	}
	if _, err := out.WriteString(syntax.ELSE.String()); err != nil {
		return fmt.Errorf("rendering condition expression ELSE token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering condition expression space: %w", err)
	}
	if err := expr(out, input.False, opts); err != nil {
		return fmt.Errorf("rendering condition expression False: %w", err)
	}

	return nil
}

func dictEntry(out io.StringWriter, input *syntax.DictEntry, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering dict entry: nil input")
	}
	if err := expr(out, input.Key, opts); err != nil {
		return fmt.Errorf("rendering dict entry Key: %w", err)
	}
	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering dict entry COLON token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering dict entry space: %w", err)
	}
	if err := expr(out, input.Value, opts); err != nil {
		return fmt.Errorf("rendering dict entry Value: %w", err)
	}

	return nil
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

	if _, err := out.WriteString(syntax.LBRACE.String()); err != nil {
		return fmt.Errorf("rendering dict expression LBRACE token: %w", err)
	}

	if err := exprSequence(out, input.List, renderOption(opts.dictOption), opts); err != nil {
		return fmt.Errorf("rendering dict expression: %w", err)
	}

	if _, err := out.WriteString(syntax.RBRACE.String()); err != nil {
		return fmt.Errorf("rendering dict expression RBRACE token: %w", err)
	}

	return nil
}

func dotExpr(out io.StringWriter, input *syntax.DotExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering dot expression: nil input")
	}

	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering dot expression X: %w", err)
	}
	if _, err := out.WriteString(syntax.DOT.String()); err != nil {
		return fmt.Errorf("rendering dot expression DOT token: %w", err)
	}
	if err := expr(out, input.Name, opts); err != nil {
		return fmt.Errorf("rendering dot expression Name: %w", err)
	}

	return nil
}

func ident(out io.StringWriter, input *syntax.Ident, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering ident: nil input")
	}

	if _, err := out.WriteString(input.Name); err != nil {
		return fmt.Errorf("rendering ident Name: %w", err)
	}

	return nil
}

func indexExpr(out io.StringWriter, input *syntax.IndexExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering index expression: nil input")
	}

	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering index expression X: %w", err)
	}
	if _, err := out.WriteString(syntax.LBRACK.String()); err != nil {
		return fmt.Errorf("rendering index expression LBRACK token: %w", err)
	}
	if err := expr(out, input.Y, opts); err != nil {
		return fmt.Errorf("rendering index expression Y: %w", err)
	}
	if _, err := out.WriteString(syntax.RBRACK.String()); err != nil {
		return fmt.Errorf("rendering index expression RBRACK token: %w", err)
	}

	return nil
}

func listExpr(out io.StringWriter, input *syntax.ListExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering list expression: nil input")
	}
	if _, err := out.WriteString(syntax.LBRACK.String()); err != nil {
		return fmt.Errorf("rendering list expression LBRACK token: %w", err)
	}

	if err := exprSequence(out, input.List, renderOption(opts.listOption), opts); err != nil {
		return fmt.Errorf("rendering list expression: %w", err)
	}

	if _, err := out.WriteString(syntax.RBRACK.String()); err != nil {
		return fmt.Errorf("rendering list expression RBRACK token: %w", err)
	}

	return nil
}

func literal(out io.StringWriter, input *syntax.Literal, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering literal: nil input")
	}

	if input.Value == nil {
		if _, err := out.WriteString(input.Raw); err != nil {
			return fmt.Errorf("rendering literal raw value: %w", err)
		}
		return nil
	}

	switch t := input.Value.(type) {
	case string:
		// starlark.String(...).String() uses strconv.Quote, which performs
		// additional allocations.
		//
		// Use a pre-allocated buffer to quote-escape the string, and an
		// unsafe.Pointer trick from strings.Builder to avoid allocation
		// when converting the byte slice to string.

		// check if the capacity is enough
		if cap(opts.stringBuffer) < len(t)*2 {
			opts.stringBuffer = make([]byte, 0, len(t)*3)
		} else {
			// reset the length if needed
			if len(opts.stringBuffer) > 0 {
				opts.stringBuffer = opts.stringBuffer[:0]
			}
		}
		opts.stringBuffer = strconv.AppendQuote(opts.stringBuffer, t)
		if _, err := out.WriteString(*(*string)(unsafe.Pointer(&opts.stringBuffer))); err != nil {
			return fmt.Errorf("rendering literal string value: %w", err)
		}
		return nil
	case int:
		if _, err := out.WriteString(starlark.MakeInt(t).String()); err != nil {
			return fmt.Errorf("rendering literal int value: %w", err)
		}
		return nil
	case uint:
		if _, err := out.WriteString(starlark.MakeUint(t).String()); err != nil {
			return fmt.Errorf("rendering literal uint value: %w", err)
		}
		return nil
	case int64:
		if _, err := out.WriteString(starlark.MakeInt64(t).String()); err != nil {
			return fmt.Errorf("rendering literal int64 value: %w", err)
		}
		return nil
	case uint64:
		if _, err := out.WriteString(starlark.MakeUint64(t).String()); err != nil {
			return fmt.Errorf("rendering literal uint64 value: %w", err)
		}
		return nil
	case *big.Int:
		if t == nil {
			return errors.New("nil literal *big.Int value provided")
		}
		if _, err := out.WriteString(starlark.MakeBigInt(t).String()); err != nil {
			return fmt.Errorf("rendering literal *big.Int value: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported literal value type %T, expected string, int, int64, uint, uint64 or *big.Int", t)
	}
}

func parenExpr(out io.StringWriter, input *syntax.ParenExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering paren expression: nil input")
	}

	if _, err := out.WriteString(syntax.LPAREN.String()); err != nil {
		return fmt.Errorf("rendering paren expression LPAREN token: %w", err)
	}
	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering paren expression X: %w", err)
	}
	if _, err := out.WriteString(syntax.RPAREN.String()); err != nil {
		return fmt.Errorf("rendering paren expression RPAREN token: %w", err)
	}

	return nil
}

func sliceExpr(out io.StringWriter, input *syntax.SliceExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering slice expression: nil input")
	}

	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering slice expression X: %w", err)
	}
	if _, err := out.WriteString(syntax.LBRACK.String()); err != nil {
		return fmt.Errorf("rendering slice expression LBRACK token: %w", err)
	}

	if input.Lo != nil {
		if err := expr(out, input.Lo, opts); err != nil {
			return fmt.Errorf("rendering slice expression Lo: %w", err)
		}
	}

	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering slice expression COLON token: %w", err)
	}

	if input.Hi != nil {
		if err := expr(out, input.Hi, opts); err != nil {
			return fmt.Errorf("rendering slice expression Hi: %w", err)
		}
	}
	if input.Step != nil {
		if _, err := out.WriteString(syntax.COLON.String()); err != nil {
			return fmt.Errorf("rendering slice expression COLON token: %w", err)
		}
		if err := expr(out, input.Step, opts); err != nil {
			return fmt.Errorf("rendering slice expression Step: %w", err)
		}
	}

	if _, err := out.WriteString(syntax.RBRACK.String()); err != nil {
		return fmt.Errorf("rendering slice expression RBRACK token: %w", err)
	}

	return nil
}

func tupleExpr(out io.StringWriter, input *syntax.TupleExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering tuple expression: nil input")
	}

	if err := exprSequence(out, input.List, renderOption(opts.tupleOption), opts); err != nil {
		return fmt.Errorf("rendering tuple expression: %w", err)
	}

	return nil
}

func unaryExpr(out io.StringWriter, input *syntax.UnaryExpr, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering unary expression: nil input")
	}

	// from the go.starlark.net docs:
	//
	// As a special case, UnaryOp{Op:Star} may also represent
	// the star parameter in def f(*args) or def f(*, x).
	if input.X == nil && input.Op != syntax.STAR {
		return fmt.Errorf("rendering unary expression, nil X value for %q token", input.Op)
	}

	if _, err := out.WriteString(input.Op.String()); err != nil {
		return fmt.Errorf("rendering unary expression, writing %q token: %w", input.Op, err)
	}

	if input.X != nil {
		if err := expr(out, input.X, opts); err != nil {
			return fmt.Errorf("rendering unary expression X: %w", err)
		}
	}

	return nil
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
