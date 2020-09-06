package starlarkgen

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.starlark.net/syntax"
)

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

	return render(out, "rendering assignment statement", opts,
		indentItem,
		exprItem(input.LHS, "LHS"),
		spaceItem,
		tokenItem(input.Op, "Op"),
		spaceItem,
		exprItem(input.RHS, "RHS"),
		newlineItem,
	)
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

	return render(out, "rendering branch statement", opts,
		indentItem,
		tokenItem(input.Token, "Token"),
		newlineItem,
	)
}

func defStmt(out io.StringWriter, input *syntax.DefStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering def statement: nil input")
	}

	items := []item{
		indentItem,
		tokenItem(syntax.DEF, "DEF"),
		spaceItem,
		exprItem(input.Name, "Name"),
		tokenItem(syntax.LPAREN, "LPAREN"),
	}
	var sep []item
	for i, param := range input.Params {
		items = append(items,
			sep...)
		items = append(items,
			exprItem(param, fmt.Sprintf("param %d", i)),
		)
		sep = commaSpace
	}
	items = append(items,
		tokenItem(syntax.RPAREN, "RPAREN"),
		colonItem,
		newlineItem,
		stmtsItem(input.Body, "Body", true),
	)
	return render(out, "rendering def statement", opts, items...)
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
			// if the literal was obtained from the parser, the whitespace might
			// be present before the token, use position to strip it
			var stripPrefix string

			// .Col value is 1-based
			if lt.Token == syntax.STRING && lt.TokenPos.Col > 1 {
				stripPrefix = strings.Repeat(" ", int(lt.TokenPos.Col-1))
			}

			lines := strings.Split(strings.ReplaceAll(strValue, `"""`, `\"\"\"`), "\n")
			items := []item{
				indentItem,
				quoteItem,
				quoteItem,
				quoteItem,
				stringItem(lines[0], "docstring line 1"),
			}
			for i := 1; i < len(lines); i++ {
				line := lines[i]
				if stripPrefix != "" && strings.HasPrefix(line, stripPrefix) {
					line = line[len(stripPrefix):]
				}
				if len(line) > 0 || i == len(lines)-1 {
					items = append(items,
						newlineItem,
						indentItem,
						stringItem(line, fmt.Sprintf("docstring line %d", i+1)),
					)
				} else {
					// do not add extra indent spaces for empty doc lines, unless it's a last one
					items = append(items,
						newlineItem,
					)
				}
			}
			items = append(items,
				quoteItem,
				quoteItem,
				quoteItem,
				newlineItem,
			)
			return render(out, "rendering expression statement", opts,
				items...,
			)
		}
	}

	return render(out, "rendering expression statement", opts,
		indentItem,
		exprItem(input.X, "X"),
		newlineItem,
	)
}

func forStmt(out io.StringWriter, input *syntax.ForStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering for statement: nil input")
	}

	return render(out, "rendering for statement", opts,
		indentItem,
		tokenItem(syntax.FOR, "FOR"),
		spaceItem,
		exprItem(input.Vars, "Vars"),
		spaceItem,
		tokenItem(syntax.IN, "IN"),
		spaceItem,
		exprItem(input.X, "X"),
		colonItem,
		newlineItem,
		stmtsItem(input.Body, "Body", true),
	)
}

func ifStmt(out io.StringWriter, input *syntax.IfStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering if statement: nil input")
	}

	items := []item{
		indentItem,
		tokenItem(syntax.IF, "IF"),
		spaceItem,
		exprItem(input.Cond, "Cond"),
		colonItem,
		newlineItem,
		stmtsItem(input.True, "True", true),
	}
	if len(input.False) > 0 {
		items = append(items,
			indentItem,
			tokenItem(syntax.ELSE, "ELSE"),
			colonItem,
			newlineItem,
			stmtsItem(input.False, "False", true),
		)
	}

	return render(out, "rendering if statement", opts, items...)
}

func loadStmt(out io.StringWriter, input *syntax.LoadStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering load statement: nil input")
	}

	if len(input.From) != len(input.To) {
		return fmt.Errorf("rendering load statement, lengths mismatch, From: %d, To: %d", len(input.From), len(input.To))
	}
	items := []item{
		indentItem,
		tokenItem(syntax.LOAD, "LOAD"),
		tokenItem(syntax.LPAREN, "LPAREN"),
		exprItem(input.Module, "Module"),
	}
	for i, elem := range input.From {
		items = append(items,
			tokenItem(syntax.COMMA, "COMMA"),
			spaceItem,
		)
		if input.To[i] != nil && input.To[i].Name != elem.Name {
			items = append(items,
				exprItem(input.To[i], fmt.Sprintf("To[%d]", i)),
			)
			// spaces around "=" if the option is set
			if opts.spaceEqBinary {
				items = append(items,
					spaceItem,
					tokenItem(syntax.EQ, "EQ"),
					spaceItem,
				)
			} else {
				items = append(items,
					tokenItem(syntax.EQ, "EQ"),
				)
			}
		}
		items = append(items,
			quoteItem,
			exprItem(elem, fmt.Sprintf("From[%d]", i)),
			quoteItem,
		)
	}
	items = append(items,
		tokenItem(syntax.RPAREN, "RPAREN"),
		newlineItem,
	)
	return render(out, "rendering load statement", opts, items...)
}

func returnStmt(out io.StringWriter, input *syntax.ReturnStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering return statement: nil input")
	}

	items := []item{
		indentItem,
		tokenItem(syntax.RETURN, "RETURN"),
	}

	if input.Result != nil {
		items = append(items,
			spaceItem,
			exprItem(input.Result, "Result"),
		)
	}

	items = append(items,
		newlineItem,
	)
	return render(out, "rendering return statement", opts, items...)
}

func whileStmt(out io.StringWriter, input *syntax.WhileStmt, opts *outputOpts) error {
	if input == nil {
		return errors.New("rendering while statement: nil input")
	}

	return render(out, "rendering if statement", opts,
		indentItem,
		tokenItem(syntax.WHILE, "WHILE"),
		spaceItem,
		exprItem(input.Cond, "Cond"),
		colonItem,
		newlineItem,
		stmtsItem(input.Body, "Body", true),
	)
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
