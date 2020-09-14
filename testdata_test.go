package starlarkgen

import (
	"io/ioutil"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

var testSources = map[string][]Option{
	"testdata/import.star":       nil,
	"testdata/test_input_1.star": {WithSpaceEqBinary(true)},
	"testdata/test_input_2.star": {
		WithSpaceEqBinary(true),
		WithCallOption(CallOptionMultilineMultipleCommaTwoAndMore),
		WithDictOption(DictOptionMultilineMultipleComma),
		WithListOption(ListOptionMultilineMultipleComma),
		WithTupleOption(TupleOptionSingleLineComma),
	},
}

func Test_testdata(t *testing.T) {
	for sf, opts := range testSources {
		t.Run(sf, func(b *testing.T) {
			tf, err := ioutil.ReadFile(sf)
			if err != nil {
				t.Fatal("error reading test file", err)
			}
			want := string(tf)
			f, err := syntax.Parse(sf, nil, 0)
			if err != nil {
				t.Fatal("error parsing test file", err)
			}

			var sb strings.Builder
			sep := ""
			for _, s := range f.Stmts {
				sb.WriteString(sep)
				d, err := StarlarkStmt(s, opts...)
				if err != nil {
					t.Fatal("error processing statement", err)
				}
				sb.WriteString(d)
				sep = "\n"
			}
			if got := sb.String(); want != got {
				t.Errorf("output mismatch, want %q, got %q", want, got)
			}
		})
	}
}

func Benchmark_testData(b *testing.B) {
	var (
		sourceMap = make(map[string]*syntax.File, len(testSources))
		err       error
	)
	// pre-parse the source files to exclude from benchmark wall clock time
	for sf := range testSources {
		sourceMap[sf], err = syntax.Parse(sf, nil, 0)
		if err != nil {
			b.Fatal("error parsing test file", err)
		}
	}
	for sf, opts := range testSources {
		b.Run(sf, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, s := range sourceMap[sf].Stmts {
					_, err := StarlarkStmt(s, opts...)

					if err != nil {
						b.Fatal("error processing statement", err)
					}
				}
			}
		})
	}
}
