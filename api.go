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
	dictOption    DictOption
	listOption    ListOption
	callOption    CallOption
	tupleOption   TupleOption
}

// copy the options, will panic on nil argument
func (o *outputOpts) copy() *outputOpts {
	c := *o
	return &c
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
	dictOption:    DictOptionSingleLine,
	listOption:    ListOptionSingleLine,
	callOption:    CallOptionSingleLine,
	tupleOption:   TupleOptionSingleLine,
}

type (
	renderOption  uint8
	multiLineType uint8
	lastCommaType uint8
)

const (
	singleLine multiLineType = iota
	multiLineMultiple
	multiLine
)

const (
	noLastComma lastCommaType = iota
	alwaysLastComma
	lastCommaTwoAndMore
)

func (ro renderOption) commaType() lastCommaType {
	return lastCommaType(uint8(ro) % 3)
}

func (ro renderOption) multiLineType() multiLineType {
	return multiLineType(uint8(ro) / 3)
}

// CallOption controls how the function calls are rendered. See examples for
// details on each specific option.
type CallOption renderOption

const (
	// CallOptionSingleLine is the default, render as single line.
	CallOptionSingleLine CallOption = iota
	// CallOptionSingleLineComma will render call as single line, with comma after last argument.
	CallOptionSingleLineComma
	// CallOptionSingleLineCommaTwoAndMore will render call as single line, with comma after last argument, if there are two or more arguments.
	CallOptionSingleLineCommaTwoAndMore

	// CallOptionMultilineMultiple will render single argument calls as single line,
	// and two and more arguments as multiline.
	CallOptionMultilineMultiple
	// CallOptionMultilineMultipleComma will render single argument calls as single line,
	// and two and more arguments as multiline, with comma after last argument.
	CallOptionMultilineMultipleComma
	// CallOptionMultilineMultipleCommaTwoAndMore will render single argument calls as single line,
	// and two and more arguments as multiline, with comma after last argument if two or more argument are present.
	CallOptionMultilineMultipleCommaTwoAndMore

	// CallOptionMultiline will render call as multiline.
	CallOptionMultiline
	// CallOptionMultilineComma will render call as multiline, with comma after last argument.
	CallOptionMultilineComma
	// CallOptionMultilineCommaTwoAndMore will render call as multiline, with comma after last argument, if there are two or more argument.
	CallOptionMultilineCommaTwoAndMore

	callOptionMax
)

// DictOption controls how the dict literals are rendered. See examples for
// details on each specific option.
type DictOption renderOption

const (
	// DictOptionSingleLine is the default, render as single line.
	DictOptionSingleLine DictOption = iota
	// DictOptionSingleLineComma will render dict as single line, with comma after last item.
	DictOptionSingleLineComma
	// DictOptionSingleLineCommaTwoAndMore will render dict as single line, with comma after last item, if there are two or more items.
	DictOptionSingleLineCommaTwoAndMore

	// DictOptionMultilineMultiple will render single element dictionaries as single line,
	// and two and more elements as multiline.
	DictOptionMultilineMultiple
	// DictOptionMultilineMultipleComma will render single element dictionaries as single line,
	// and two and more elements as multiline, with comma after last item.
	DictOptionMultilineMultipleComma
	// DictOptionMultilineMultipleCommaTwoAndMore will render single element dictionaries as single line,
	// and two and more elements as multiline, with comma after last item if two or more elements are present.
	DictOptionMultilineMultipleCommaTwoAndMore

	// DictOptionMultiline will render dict as multiline.
	DictOptionMultiline
	// DictOptionMultilineComma will render dict as multiline, with comma after last item.
	DictOptionMultilineComma
	// DictOptionMultilineCommaTwoAndMore will render dict as multiline, with comma after last item, if there are two or more items.
	DictOptionMultilineCommaTwoAndMore

	dictOptionMax
)

// ListOption controls how the list literals are rendered. See examples for
// details on each specific option.
type ListOption renderOption

const (
	// ListOptionSingleLine is the default, render as single line.
	ListOptionSingleLine ListOption = iota
	// ListOptionSingleLineComma will render list as single line, with comma after last item.
	ListOptionSingleLineComma
	// ListOptionSingleLineCommaTwoAndMore will render list as single line, with comma after last item, if there are two or more items.
	ListOptionSingleLineCommaTwoAndMore

	// ListOptionMultilineMultiple will render single element lists as single line,
	// and two and more elements as multiline.
	ListOptionMultilineMultiple
	// ListOptionMultilineMultipleComma will render single element lists as single line,
	// and two and more elements as multiline, with comma after last item.
	ListOptionMultilineMultipleComma
	// ListOptionMultilineMultipleCommaTwoAndMore will render single element lists as single line,
	// and two and more elements as multiline, with comma after last item if two or more elements are present.
	ListOptionMultilineMultipleCommaTwoAndMore

	// ListOptionMultiline will render list as multiline.
	ListOptionMultiline
	// ListOptionMultilineComma will render list as multiline, with comma after last item.
	ListOptionMultilineComma
	// ListOptionMultilineCommaTwoAndMore will render list as multiline, with comma after last item, if there are two or more items.
	ListOptionMultilineCommaTwoAndMore

	listOptionMax
)

// TupleOption controls how the tuple literals are rendered. See examples for
// details on each specific option.
type TupleOption renderOption

const (
	// TupleOptionSingleLine is the default, render as single line.
	TupleOptionSingleLine TupleOption = iota
	// TupleOptionSingleLineComma will render tuple as single line, with comma after last item.
	TupleOptionSingleLineComma
	// TupleOptionSingleLineCommaTwoAndMore will render tuple as single line, with comma after last item, if there are two or more items.
	TupleOptionSingleLineCommaTwoAndMore

	// TupleOptionMultilineMultiple will render single element tuples as single line,
	// and two and more elements as multiline.
	TupleOptionMultilineMultiple
	// TupleOptionMultilineMultipleComma will render single element tuples as single line,
	// and two and more elements as multiline, with comma after last item.
	TupleOptionMultilineMultipleComma
	// TupleOptionMultilineMultipleCommaTwoAndMore will render single element tuples as single line,
	// and two and more elements as multiline, with comma after last item if two or more elements are present.
	TupleOptionMultilineMultipleCommaTwoAndMore

	// TupleOptionMultiline will render tuple as multiline.
	TupleOptionMultiline
	// TupleOptionMultilineComma will render tuple as multiline, with comma after last item.
	TupleOptionMultilineComma
	// TupleOptionMultilineCommaTwoAndMore will render tuple as multiline, with comma after last item, if there are two or more items.
	TupleOptionMultilineCommaTwoAndMore

	tupleOptionMax
)

// Option represents Starlark code rendering option.
type Option func(*outputOpts) (*outputOpts, error)

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

// WithCallOption sets the option to render function calls.
func WithCallOption(value CallOption) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		if value >= callOptionMax {
			return nil, fmt.Errorf("invalid option value %v", value)
		}
		c := o.copy()
		c.callOption = value
		return c, nil
	}
}

// WithDictOption sets the option to render dictionary literals.
func WithDictOption(value DictOption) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		if value >= dictOptionMax {
			return nil, fmt.Errorf("invalid option value %v", value)
		}
		c := o.copy()
		c.dictOption = value
		return c, nil
	}
}

// WithListOption sets the option to render list literals.
func WithListOption(value ListOption) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		if value >= listOptionMax {
			return nil, fmt.Errorf("invalid option value %v", value)
		}
		c := o.copy()
		c.listOption = value
		return c, nil
	}
}

// WithTupleOption sets the option to render tuple literals.
func WithTupleOption(value TupleOption) Option {
	return func(o *outputOpts) (*outputOpts, error) {
		if value >= tupleOptionMax {
			return nil, fmt.Errorf("invalid option value %v", value)
		}
		c := o.copy()
		c.tupleOption = value
		return c, nil
	}
}

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
