package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// код писать тут

func main() {
	// if len(os.Args) < 2 {
	// 	fmt.Println("usage: \n\tcodegen 'path to file'")
	// 	return
	// }
	infile := "../api.go"
	outfile := "../api_handlers.go"

	fs := token.NewFileSet()
	nodes, err := parser.ParseFile(fs, infile, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	out, err := os.Create(outfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintln(out, "package ", nodes.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "import (\n")
	fmt.Fprintf(out, "\t\"%s\"\n", "net/http")
	fmt.Fprintf(out, "\t\"%s\"\n", "context")
	fmt.Fprintf(out, "\t\"%s\"\n", "encoding/json")
	fmt.Fprintf(out, "\n)\n")
	fmt.Fprintln(out)

	mm := map[string][]MethodHandler{}

	for _, d := range nodes.Decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if f.Doc == nil {
			continue
		}
		needCodegen := false
		for _, comment := range f.Doc.List {
			needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
		}

		if !needCodegen {
			continue
		}

		if f.Recv == nil {
			continue
		}

		mh, e := ParseMethodHandler(f)
		if e != nil {
			fmt.Println(e)
			break
		}

		mm[mh.rec] = append(mm[mh.rec], *mh)

		mh.PrintHandler(out)
	}
	MakeHTTPHandler(out, &mm)
}
