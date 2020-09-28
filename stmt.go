package starlarkgen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unsafe"

	"go.starlark.net/syntax"
)

var (
	tripleQuote = strings.Repeat(quote, 3)
)

func stmtSequence(out io.StringWriter, input []syntax.Stmt, opts *outputOpts) error {
	stOpts := opts.addDepth(1)
	for ii, st := range input {
		if err := stmt(out, st, stOpts); err != nil {
			return fmt.Errorf("statement index %d: %w", ii, err)
		}
	}
	return nil
}

func hasSpacePrefix(buf []byte, l int) bool {
	if len(buf) < l {
		return false
	}
	for i, r := range buf {
		if i == l {
			return true
		}
		if r != ' ' {
			return false
		}
	}

	return true
}

func assignStmt(out io.StringWriter, input *syntax.AssignStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering assign statement: nil input")
	}

	// check if Op token is supported for assignment
	// valid tokens are: EQ | {PLUS,MINUS,STAR,PERCENT}_EQ
	switch input.Op {
	case syntax.EQ, syntax.PLUS_EQ, syntax.MINUS_EQ, syntax.STAR_EQ, syntax.PERCENT_EQ:
	default:
		return fmt.Errorf("rendering assign statement: unsupported Op token %v, expected one of: %v, %v, %v, %v, %v", input.Op, syntax.EQ, syntax.PLUS_EQ, syntax.MINUS_EQ, syntax.STAR_EQ, syntax.PERCENT_EQ)
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering assignment statement indent: %w", err)
	}
	if err := expr(out, input.LHS, opts); err != nil {
		return fmt.Errorf("rendering assignment statement LHS: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering assignment statement space: %w", err)
	}
	if _, err := out.WriteString(input.Op.String()); err != nil {
		return fmt.Errorf("rendering assignment statement Op token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering assignment statement space: %w", err)
	}
	if err := expr(out, input.RHS, opts); err != nil {
		return fmt.Errorf("rendering assignment statement RHS: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering assignment statement NEWLINE token: %w", err)
	}

	return nil
}

func branchStmt(out io.StringWriter, input *syntax.BranchStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering branch statement: nil input")
	}

	// check if the token is supported for this type
	// valid tokens are: BREAK | CONTINUE | PASS
	switch input.Token {
	case syntax.BREAK, syntax.CONTINUE, syntax.PASS:
	default:
		return fmt.Errorf("rendering branch statement: unsupported token %v, expected %v, %v or %v", input.Token, syntax.BREAK, syntax.CONTINUE, syntax.PASS)
	}
	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering branch statement indent: %w", err)
	}
	if _, err := out.WriteString(input.Token.String()); err != nil {
		return fmt.Errorf("rendering branch statement Token token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering branch statement NEWLINE token: %w", err)
	}
	return nil
}

func defStmt(out io.StringWriter, input *syntax.DefStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering def statement: nil input")
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering def statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.DEF.String()); err != nil {
		return fmt.Errorf("rendering def statement DEF token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering def statement space: %w", err)
	}
	if err := expr(out, input.Name, opts); err != nil {
		return fmt.Errorf("rendering def statement Name: %w", err)
	}
	if _, err := out.WriteString(syntax.LPAREN.String()); err != nil {
		return fmt.Errorf("rendering def statement LPAREN token: %w", err)
	}
	// TODO: add def rendering options
	if err := exprSequence(out, input.Params, renderOption(0), opts); err != nil {
		return fmt.Errorf("rendering def statement Params: %w", err)
	}
	if _, err := out.WriteString(syntax.RPAREN.String()); err != nil {
		return fmt.Errorf("rendering def statement RPAREN token: %w", err)
	}
	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering def statement COLON token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering def statement NEWLINE token: %w", err)
	}
	if err := stmtSequence(out, input.Body, opts); err != nil {
		return fmt.Errorf("rendering def statement Body: %w", err)
	}
	return nil
}

func docstring(out io.StringWriter, input *syntax.Literal, strValue string, opts *outputOpts) error {
	// if the literal was obtained from the parser, the whitespace might
	// be present before the token, use position to strip it
	var stripPrefix int32

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering docstring expression statement indent: %w", err)
	}
	if _, err := out.WriteString(tripleQuote); err != nil {
		return fmt.Errorf("rendering docstring expression statement TRIPLE QUOTE token: %w", err)
	}

	// .Col value is 1-based
	if input.Token == syntax.STRING && input.TokenPos.Col > 1 {
		stripPrefix = input.TokenPos.Col - 1
	}

	if cap(opts.stringBuffer) < len(strValue)*2 {
		opts.stringBuffer = make([]byte, 0, len(strValue)*3)
	} else {
		// reset the length if needed
		if len(opts.stringBuffer) > 0 {
			opts.stringBuffer = opts.stringBuffer[:0]
		}
	}
	for i := 0; i < len(strValue); i++ {
		if i <= len(strValue)-3 {
			if b := strValue[i : i+3]; b == `"""` {
				opts.stringBuffer = append(opts.stringBuffer, `\"\"\"`...)
				i += 2
				continue
			}
		}
		opts.stringBuffer = append(opts.stringBuffer, strValue[i])
	}

	var lineNum int

	for pos := 0; pos < len(opts.stringBuffer); {
		var buf []byte
		// do not use bufio.Scanner to avoid extra 4kb allocation
		if i := bytes.IndexByte(opts.stringBuffer[pos:], '\n'); i >= 0 {
			buf = opts.stringBuffer[pos : pos+i]
			pos += i + 1
		} else {
			buf = opts.stringBuffer[pos:]
			pos = len(opts.stringBuffer)
		}

		if lineNum == 0 {
			if _, err := out.WriteString(*(*string)(unsafe.Pointer(&buf))); err != nil {
				return fmt.Errorf("rendering docstring expression statement: docstring line 1: %w", err)
			}
			lineNum++
			continue
		}
		if stripPrefix > 0 && hasSpacePrefix(buf, int(stripPrefix)) {
			buf = buf[stripPrefix:]
		}

		if _, err := out.WriteString(newline); err != nil {
			return fmt.Errorf("rendering docstring expression statement NEWLINE token: %w", err)
		}
		if len(buf) > 0 || pos >= len(opts.stringBuffer) {
			if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
				return fmt.Errorf("rendering docstring expression statement indent: %w", err)
			}
			if _, err := out.WriteString(*(*string)(unsafe.Pointer(&buf))); err != nil {
				return fmt.Errorf("rendering docstring expression statement: docstring line %d: %w", lineNum+1, err)
			}
		}
		lineNum++
	}
	// if the last line is empty, still write the indent
	if len(opts.stringBuffer) > 0 && opts.stringBuffer[len(opts.stringBuffer)-1] == '\n' {
		if _, err := out.WriteString(newline); err != nil {
			return fmt.Errorf("rendering docstring expression statement NEWLINE token: %w", err)
		}
		if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
			return fmt.Errorf("rendering docstring expression statement indent: %w", err)
		}
	}

	if _, err := out.WriteString(tripleQuote); err != nil {
		return fmt.Errorf("rendering docstring expression statement TRIPLE QUOTE token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering docstring expression statement NEWLINE token: %w", err)
	}

	return nil
}

func exprStmt(out io.StringWriter, input *syntax.ExprStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering expression statement: nil input")
	}

	// special case: docstring render, e.g.
	//
	// def foo_bar():
	//     """some line 1
	//     line 2
	//     """
	if lt, ok := input.X.(*syntax.Literal); ok {
		if strValue, ok := lt.Value.(string); ok {
			return docstring(out, lt, strValue, opts)
		}
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering expression statement indent: %w", err)
	}
	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering expression statement X: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering expression statement NEWLINE token: %w", err)
	}

	return nil
}

func forStmt(out io.StringWriter, input *syntax.ForStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering for statement: nil input")
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering for statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.FOR.String()); err != nil {
		return fmt.Errorf("rendering for statement FOR token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering for statement space: %w", err)
	}
	if err := expr(out, input.Vars, opts); err != nil {
		return fmt.Errorf("rendering for statement Vars: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering for statement space: %w", err)
	}
	if _, err := out.WriteString(syntax.IN.String()); err != nil {
		return fmt.Errorf("rendering for statement IN token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering for statement space: %w", err)
	}
	if err := expr(out, input.X, opts); err != nil {
		return fmt.Errorf("rendering for statement X: %w", err)
	}
	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering for statement COLON token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering for statement NEWLINE token: %w", err)
	}
	if err := stmtSequence(out, input.Body, opts); err != nil {
		return fmt.Errorf("rendering for statement Body: %w", err)
	}

	return nil
}

func ifStmt(out io.StringWriter, input *syntax.IfStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering if statement: nil input")
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering if statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.IF.String()); err != nil {
		return fmt.Errorf("rendering if statement IF token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering if statement space: %w", err)
	}
	if err := expr(out, input.Cond, opts); err != nil {
		return fmt.Errorf("rendering if statement Cond: %w", err)
	}
	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering if statement COLON token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering if statement NEWLINE token: %w", err)
	}
	if err := stmtSequence(out, input.True, opts); err != nil {
		return fmt.Errorf("rendering if statement True: %w", err)
	}

	if len(input.False) > 0 {
		if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
			return fmt.Errorf("rendering if statement indent: %w", err)
		}
		if _, err := out.WriteString(syntax.ELSE.String()); err != nil {
			return fmt.Errorf("rendering if statement ELSE token: %w", err)
		}
		if _, err := out.WriteString(syntax.COLON.String()); err != nil {
			return fmt.Errorf("rendering if statement COLON token: %w", err)
		}
		if _, err := out.WriteString(newline); err != nil {
			return fmt.Errorf("rendering if statement NEWLINE token: %w", err)
		}
		if err := stmtSequence(out, input.False, opts); err != nil {
			return fmt.Errorf("rendering if statement False: %w", err)
		}
	}

	return nil
}

func loadStmt(out io.StringWriter, input *syntax.LoadStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering load statement: nil input")
	}
	if len(input.From) != len(input.To) {
		return fmt.Errorf("rendering load statement, lengths mismatch, From: %d, To: %d", len(input.From), len(input.To))
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering load statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.LOAD.String()); err != nil {
		return fmt.Errorf("rendering load statement LOAD token: %w", err)
	}
	if _, err := out.WriteString(syntax.LPAREN.String()); err != nil {
		return fmt.Errorf("rendering load statement LPAREN token: %w", err)
	}
	if err := expr(out, input.Module, opts); err != nil {
		return fmt.Errorf("rendering load statement Module: %w", err)
	}

	for i, elem := range input.From {
		// load statement must import at least 1 symbol
		if _, err := out.WriteString(syntax.COMMA.String()); err != nil {
			return fmt.Errorf("rendering load statement COMMA token: %w", err)
		}
		if _, err := out.WriteString(space); err != nil {
			return fmt.Errorf("rendering load statement space: %w", err)
		}
		if input.To[i] != nil && input.To[i].Name != elem.Name {
			if err := expr(out, input.To[i], opts); err != nil {
				return fmt.Errorf("rendering load statement To[%d]: %w", i, err)
			}
			// spaces around "=" if the option is set
			if opts.spaceEqBinary {
				if _, err := out.WriteString(space); err != nil {
					return fmt.Errorf("rendering load statement space: %w", err)
				}
				if _, err := out.WriteString(syntax.EQ.String()); err != nil {
					return fmt.Errorf("rendering load statement EQ token: %w", err)
				}
				if _, err := out.WriteString(space); err != nil {
					return fmt.Errorf("rendering load statement space: %w", err)
				}
			} else {
				if _, err := out.WriteString(syntax.EQ.String()); err != nil {
					return fmt.Errorf("rendering load statement EQ token: %w", err)
				}
			}
		}
		if _, err := out.WriteString(quote); err != nil {
			return fmt.Errorf("rendering load statement QUOTE token: %w", err)
		}
		if err := expr(out, elem, opts); err != nil {
			return fmt.Errorf("rendering load statement From[%d]: %w", i, err)
		}
		if _, err := out.WriteString(quote); err != nil {
			return fmt.Errorf("rendering load statement QUOTE token: %w", err)
		}
	}

	if _, err := out.WriteString(syntax.RPAREN.String()); err != nil {
		return fmt.Errorf("rendering load statement RPAREN token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering load statement NEWLINE token: %w", err)
	}

	return nil
}

func returnStmt(out io.StringWriter, input *syntax.ReturnStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering return statement: nil input")
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering return statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.RETURN.String()); err != nil {
		return fmt.Errorf("rendering return statement RETURN token: %w", err)
	}

	if input.Result != nil {
		if _, err := out.WriteString(space); err != nil {
			return fmt.Errorf("rendering return statement space: %w", err)
		}
		if err := expr(out, input.Result, opts); err != nil {
			return fmt.Errorf("rendering return statement Result: %w", err)
		}
	}

	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering return statement NEWLINE token: %w", err)
	}

	return nil
}

func whileStmt(out io.StringWriter, input *syntax.WhileStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering while statement: nil input")
	}

	if err := writeRepeat(out, opts.indent, opts.depth); err != nil {
		return fmt.Errorf("rendering while statement indent: %w", err)
	}
	if _, err := out.WriteString(syntax.WHILE.String()); err != nil {
		return fmt.Errorf("rendering while statement WHILE token: %w", err)
	}
	if _, err := out.WriteString(space); err != nil {
		return fmt.Errorf("rendering while statement space: %w", err)
	}
	if err := expr(out, input.Cond, opts); err != nil {
		return fmt.Errorf("rendering while statement Cond: %w", err)
	}
	if _, err := out.WriteString(syntax.COLON.String()); err != nil {
		return fmt.Errorf("rendering while statement COLON token: %w", err)
	}
	if _, err := out.WriteString(newline); err != nil {
		return fmt.Errorf("rendering while statement NEWLINE token: %w", err)
	}
	if err := stmtSequence(out, input.Body, opts); err != nil {
		return fmt.Errorf("rendering while statement Body: %w", err)
	}

	return nil
}

func stmt(out io.StringWriter, input syntax.Stmt, opts *outputOpts) error {
	switch t := input.(type) {
	case *syntax.AssignStmt:
		return assignStmt(out, t, opts)
	case *syntax.BranchStmt:
		return branchStmt(out, t, opts)
	case *syntax.DefStmt:
		return defStmt(out, t, opts)
	case *syntax.ExprStmt:
		return exprStmt(out, t, opts)
	case *syntax.ForStmt:
		return forStmt(out, t, opts)
	case *syntax.IfStmt:
		return ifStmt(out, t, opts)
	case *syntax.LoadStmt:
		return loadStmt(out, t, opts)
	case *syntax.ReturnStmt:
		return returnStmt(out, t, opts)
	case *syntax.WhileStmt:
		return whileStmt(out, t, opts)
	default:
		return fmt.Errorf("unsupported type %T", input)
	}
}
