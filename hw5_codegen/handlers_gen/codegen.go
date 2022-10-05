package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"
)

var (
	headerTmp = template.Must(template.New("structTmpl").Parse(`
package {{ .Name }}

import (
	{{ range .Imports}}"{{.}}"
	{{ end }}
)
	`))
)

type PackageTemplate struct {
	Name    string
	Imports []string
}

type ParsedFile struct {
	Methods []*MethodHandler
	Structs []*StructDef
}

type Generator struct {
	out  io.Writer
	file *ast.File

	generatorPrefix string
	validatorPrefix string

	parsed *ParsedFile
}

func NewGenerator(in, out, gen, val string) (*Generator, error) {
	o, e := os.Create(out)
	if e != nil {
		return nil, e
	}

	fs := token.NewFileSet()
	f, e := parser.ParseFile(fs, in, nil, parser.ParseComments)
	if e != nil {
		return nil, e
	}

	return &Generator{
		out:             o,
		file:            f,
		generatorPrefix: gen,
		validatorPrefix: val,
		parsed:          &ParsedFile{},
	}, nil
}

func (g *Generator) WritePackageHeader(packages []string) {
	pt := PackageTemplate{
		Name:    g.file.Name.Name,
		Imports: packages,
	}

	headerTmp.Execute(g.out, pt)
}

func (g *Generator) ParseMethods() error {
	res, err := ParseMethods(g.file, g.generatorPrefix)
	if err != nil {
		return err
	}

	g.parsed.Methods = res

	return nil
}

func (g *Generator) ParseStructs() error {
	arguments := ""
	for _, h := range g.parsed.Methods {
		for _, m := range h.child {
			arguments += strings.Join(m.argt, " ")
		}
	}

	res, err := ParseStructs(g.file, arguments, g.validatorPrefix)
	if err != nil {
		return err
	}

	g.parsed.Structs = res

	return nil
}

func (g *Generator) WriteHttpHandlers() {
	for _, m := range g.parsed.Methods {
		httpTmpl.Execute(g.out, m.HttpTemplate())
		for _, h := range m.child {
			hndlTmpl.Execute(g.out, h.Template())
		}
	}
}

func (g *Generator) WriteStructs() {
	for _, s := range g.parsed.Structs {
		structTmpl.Execute(g.out, s.Template())
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage:\n\tcodegen <infile> <outfile>\n")
		return
	}

	infile := os.Args[1]
	outfile := os.Args[2]

	g, err := NewGenerator(infile, outfile, "apigen", "apivalidator")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = g.ParseMethods()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = g.ParseStructs()
	if err != nil {
		fmt.Println(err)
		return
	}

	g.WritePackageHeader([]string{"encoding/json", "fmt", "io/ioutil", "net/http", "net/url", "strconv", "strings"})
	g.WriteHttpHandlers()
	g.WriteStructs()
}
