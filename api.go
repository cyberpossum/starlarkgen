package starlarkgen

import (
	"fmt"
	"io"
	"strings"

	"go.starlark.net/syntax"
)

const (
	defaultIndent        = "    "
	defaultDepth         = 0
	defaultSpaceEqBinary = false
)

type outputOpts struct {
	depth         int
	indent        string
	spaceEqBinary bool
}

func (o *outputOpts) copy() *outputOpts {
	return &outputOpts{
		depth:         o.depth,
		indent:        o.indent,
		spaceEqBinary: o.spaceEqBinary,
	}
}

func (o *outputOpts) addDepth(n int) *outputOpts {
	c := o.copy()
	c.depth += n
	return c
}

var defaultOpts = outputOpts{
	depth:         defaultDepth,
	indent:        defaultIndent,
	spaceEqBinary: defaultSpaceEqBinary,
}

// WithSpaceEqBinary sets the behavior of how the binary pairs foo=bar are rendered.
// when set to true, render results are
//   foo(bar = 1, baz = "z")
// when set to false, render results are
//   foo(bar=1, baz="z")
// This setting also applies to aliases in load(...) statement. The default value is false.
//
// This setting does not affect the behavior of assignments, which always have
// the spaces around equality sign.
func WithSpaceEqBinary(value bool) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		c := o.copy()
		c.spaceEqBinary = value
		return c, nil
	}
}

// WithDepth sets the initial indentation depth.
func WithDepth(depth int) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		if depth < 0 {
			return nil, fmt.Errorf("invalid depth value %d, value must be >= 0", depth)
		}
		c := o.copy()
		c.depth = depth
		return c, nil
	}
}

// WithIndent replaces the default indentation sequence.
func WithIndent(indent string) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		c := o.copy()
		c.indent = indent
		return c, nil
	}
}

// Option represents Starlark code rendering option.
type Option func(*outputOpts) (*outputOpts, error)

func getOutputOpts(options ...Option) (*outputOpts, error) {
	var (
		opts = &defaultOpts
		err  error
	)
	for _, o := range options {
		opts, err = o(opts)
		if err != nil {
			return nil, err
		}
	}

	return opts, nil
}

// StarlarkStmt produces Starlark source code for a single statement
// using the options supplied.
// In case of an error the string output is always empty.
func StarlarkStmt(input syntax.Stmt, options ...Option) (string, error) {
	var sb strings.Builder
	if err := WriteStmt(&sb, input, options...); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// WriteStmt writes the Starlark statement to the provided writer
// using the options supplied.
// In case of an error incomplete results might be written to the output,
// use StarlarkStmt to avoid handling partial input.
func WriteStmt(output io.StringWriter, input syntax.Stmt, options ...Option) error {
	opts, err := getOutputOpts(options...)
	if err != nil {
		return err
	}
	return stmt(output, input, opts)
}

// StarlarkExpr produces Starlark source code for a single expression
// using the options supplied.
// In case of an error the string output is always empty.
func StarlarkExpr(input syntax.Expr, options ...Option) (string, error) {
	var sb strings.Builder
	if err := WriteExpr(&sb, input, options...); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// WriteExpr writes the Starlark expression to the provided writer
// using the options supplied.
// In case of an error incomplete results might be written to the output,
// use StarlarkStmt to avoid handling partial input.
func WriteExpr(output io.StringWriter, input syntax.Expr, options ...Option) error {
	opts, err := getOutputOpts(options...)
	if err != nil {
		return err
	}
	return expr(output, input, opts)
}
