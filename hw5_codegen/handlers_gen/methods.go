package main

import (
	"fmt"
	"go/ast"
	"io"
	"strings"
	"text/template"
)

var (
	handlTpl = template.Must(template.New("handlTpl").Parse(`
func ({{.FieldArg}} {{.FieldRecv}}) handle{{.FieldName}}(w http.ResponseWriter, r *http.Request) {
	{{.FieldArg}}.{{.FieldName}}({{.FieldArgs}})
}
	`))
	httpTpl = template.Must(template.New("handlTpl").Parse(`
func ({{.FieldArg}} {{.FieldRecv}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {`))
	caseTpl = template.Must(template.New("handlTpl").Parse(`
	case "{{.FieldUrl}}": 
		{{.FieldArg}}.handle{{.FieldName}}(w, r)`))
	defaultTpl = template.Must(template.New("handlTpl").Parse(`
	default: 
		w.WriteHeader(http.StatusNotFound)`))
)

type MethodTemplate struct {
	FieldName string
	FieldArg  string
	FieldRecv string
	FieldArgs string
	FieldArgt string
	FieldRets string
	FieldRett string
	FieldUrl  string
}

type MethodHandler struct {
	n   string // handler name
	rec string // receiver
	arg string // receiver name

	args []string // methods args
	argt []string // methods args types
	rets []string // methods ret vals
	rett []string // methods ret vals types

	url string // handler url
}

func (mh *MethodHandler) Template() MethodTemplate {
	return MethodTemplate{
		FieldName: mh.n,
		FieldArg:  mh.arg,
		FieldRecv: mh.rec,
		FieldArgs: strings.Join(mh.args, ", "),
		FieldArgt: strings.Join(mh.argt, ", "),
		FieldRets: strings.Join(mh.rets, ", "),
		FieldRett: strings.Join(mh.rett, ", "),
		FieldUrl:  mh.url,
	}
}

func (mh *MethodHandler) PrintHandler(w io.Writer) {
	fmt.Fprintf(w, "func (%s %s) ", mh.arg, mh.rec)
	fmt.Fprintf(w, "handle%s(w http.ResponseWriter, r *http.Request) {\n", mh.n)
	for i := range mh.argt {
		fmt.Fprintf(w, "\tvar %s %s\n", mh.args[i], mh.argt[i])
	}
	fmt.Fprintf(w, "\t%s := ", strings.Join(mh.rett, ", "))
	fmt.Fprintf(w, "%s.%s(%s)\n", mh.arg, mh.n, strings.Join(mh.args, ", "))
	fmt.Fprintf(w, "\tjs, _ := json.Marshal(%s)\n", mh.rett[0])

	fmt.Fprintf(w, "\tw.WriteHeader(http.StatusOK)\n")
	fmt.Fprintf(w, "\tw.Write(js)")
	fmt.Fprintf(w, "\n}\n\n")
}

func ParseMethodHandler(f *ast.FuncDecl) (*MethodHandler, error) {
	rv := &MethodHandler{n: f.Name.Name}

	// signature
	for _, r := range f.Recv.List {
		rv.arg = r.Names[0].Name
		switch xv := r.Type.(type) {
		case *ast.StarExpr:
			if si, ok := xv.X.(*ast.Ident); ok {
				rv.rec = "*" + si.Name
			}
		case *ast.Ident:
		}
	}

	// arguments
	for _, r := range f.Type.Params.List {
		rv.args = append(rv.args, r.Names[0].Name)
		switch xv := r.Type.(type) {
		case *ast.StarExpr:
			if si, ok := xv.X.(*ast.Ident); ok {
				rv.argt = append(rv.argt, "*"+si.Name)
			}
		case *ast.Ident:
			rv.argt = append(rv.argt, xv.Name)
		case *ast.SelectorExpr:
			if si, ok := xv.X.(*ast.Ident); ok {
				rv.argt = append(rv.argt, si.Name+"."+xv.Sel.Name)
			}
		}
	}

	// return values
	for _, r := range f.Type.Results.List {
		switch xv := r.Type.(type) {
		case *ast.StarExpr:
			if si, ok := xv.X.(*ast.Ident); ok {
				rv.rets = append(rv.rets, "*"+si.Name)
				rv.rett = append(rv.rett, strings.ToLower(string(si.Name[0])))
			}
		case *ast.Ident:
			rv.rets = append(rv.rets, xv.Name)
			rv.rett = append(rv.rett, strings.ToLower(string(xv.Name[0])))
		}
	}

	// url
	hasurl := false
	ss := []string{}
	for _, r := range f.Doc.List {
		if strings.Contains(r.Text, "{\"url\"") {
			hasurl = true
			ss = strings.Split(r.Text, "\"")
		}
	}
	if !hasurl {
		return nil, fmt.Errorf("url is not specified")
	}
	rv.url = ss[3]

	return rv, nil
}

func MakeHTTPHandler(w io.Writer, mm *map[string][]MethodHandler) {
	for _, m := range *mm {
		httpTpl.Execute(w, m[0].Template())
		for _, i := range m {
			caseTpl.Execute(w, i.Template())
		}
		fmt.Fprintf(w, "\n\tdefault:")
		fmt.Fprintf(w, "\n\t\tw.WriteHeader(http.StatusNotFound)")
		fmt.Fprintf(w, "\n\t}")
		fmt.Fprintf(w, "\n}\n")
	}
}
