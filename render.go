package starlarkgen

import (
	"fmt"
	"io"
	"strings"

	"go.starlark.net/syntax"
)

type itemType int8

const (
	exprType itemType = iota + 1
	stmtsType
	indentType
	tokenType
	stringType
)

type item struct {
	itemType
	token     syntax.Token
	expr      syntax.Expr
	stmts     []syntax.Stmt
	addIndent bool
	value     string
	valueDesc string
}

var (
	quoteItem   = item{itemType: stringType, value: "\"", valueDesc: "quote"}
	spaceItem   = item{itemType: stringType, value: " ", valueDesc: "space"}
	colonItem   = item{itemType: tokenType, token: syntax.COLON, valueDesc: "COLON"}
	indentItem  = item{itemType: indentType}
	newlineItem = item{itemType: tokenType, token: syntax.NEWLINE, valueDesc: "NEWLINE"}

	commaSpace = []item{tokenItem(syntax.COMMA, "COMMA"), spaceItem}
)

func exprItem(expr syntax.Expr, desc string) item {
	return item{itemType: exprType, expr: expr, valueDesc: desc}
}

func stmtsItem(stmts []syntax.Stmt, desc string, addIndent bool) item {
	return item{itemType: stmtsType, stmts: stmts, valueDesc: desc, addIndent: addIndent}
}

func stringItem(value, desc string) item {
	return item{itemType: stringType, value: value, valueDesc: desc}
}

func tokenItem(value syntax.Token, desc string) item {
	return item{itemType: tokenType, token: value, valueDesc: desc}
}

func render(out io.StringWriter, errPrefix string, opts *outputOpts, items ...item) error {
	for _, i := range items {
		switch i.itemType {
		case exprType:
			if err := expr(out, i.expr, opts); err != nil {
				return fmt.Errorf("%s %s: %w", errPrefix, i.valueDesc, err)
			}
		case stmtsType:
			stOpts := opts
			if i.addIndent {
				stOpts = stOpts.addDepth(1)
			}
			for ii, st := range i.stmts {
				if err := stmt(out, st, stOpts); err != nil {
					return fmt.Errorf("%s, rendering %s statement index %d: %w", errPrefix, i.valueDesc, ii, err)
				}
			}
		case indentType:
			if _, err := out.WriteString(strings.Repeat(opts.indent, opts.depth)); err != nil {
				return fmt.Errorf("%s indent: %w", errPrefix, err)
			}
		case stringType:
			if _, err := out.WriteString(i.value); err != nil {
				return fmt.Errorf("%s %s: %w", errPrefix, i.valueDesc, err)
			}
		case tokenType:
			switch i.token {
			case syntax.ILLEGAL, syntax.EOF, syntax.INDENT, syntax.OUTDENT, syntax.IDENT, syntax.INT, syntax.FLOAT, syntax.STRING:
				return fmt.Errorf("%s %s token: %w", errPrefix, i.valueDesc, fmt.Errorf("%v not supported", i.token))
			case syntax.NEWLINE:
				if _, err := out.WriteString("\n"); err != nil {
					return fmt.Errorf("%s %s token: %w", errPrefix, i.valueDesc, err)
				}
			default:
				if _, err := out.WriteString(i.token.String()); err != nil {
					return fmt.Errorf("%s %s token: %w", errPrefix, i.valueDesc, err)
				}
			}
		default:
			return fmt.Errorf("%s: item type %d is not supported in render", errPrefix, i.itemType)
		}
	}
	return nil
}
