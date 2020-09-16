![Go](https://github.com/cyberpossum/starlarkgen/workflows/Go/badge.svg)
[![Documentation](https://godoc.org/github.com/cyberpossum/starlarkgen?status.svg)](http://godoc.org/github.com/cyberpossum/starlarkgen)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cyberpossum/starlarkgen)](https://pkg.go.dev/github.com/cyberpossum/starlarkgen)

# starlarkgen

Package `starlarkgen` provides Starlark code generation methods from [go.starlark.net](https://go.starlark.net) syntax tree primitives

# Docs and examples

Check out the docs at the usual place: on [godoc.org](https://godoc.org/github.com/cyberpossum/starlarkgen) or [pkg.go.dev](https://pkg.go.dev/github.com/cyberpossum/starlarkgen)

## Simple usage example

Add a prefix `new_` prefix for all the methods and calls by parsing the source,
walking the AST tree and rebuilding the source back.

```
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
```

[See the full example code](example_test.go)

Also see the examples in the [docs](https://godoc.org/github.com/cyberpossum/starlarkgen)