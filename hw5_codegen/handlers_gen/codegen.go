package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
)

// код писать тут

func writePackageHeader(w io.Writer, pn string) {
	fmt.Fprintln(w, "package ", pn)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "import (\n")
	fmt.Fprintf(w, "\t\"%s\"\n", "net/http")
	fmt.Fprintf(w, "\t\"%s\"\n", "context")
	fmt.Fprintf(w, "\t\"%s\"\n", "encoding/json")
	fmt.Fprintf(w, "\n)\n")
	fmt.Fprintln(w)
}

func main() {
	outfile := "../api_handlers.go"
	out, err := os.Create(outfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	infile := "../api.go"
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, infile, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	writePackageHeader(out, file.Name.Name)

	// find all methods handlers
	mhs, err := ProccessFuncDecls(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	WriteHttpHandlersFormat(out, mhs)

	_, err = ParseStructs(file, "CreateParams")
}
