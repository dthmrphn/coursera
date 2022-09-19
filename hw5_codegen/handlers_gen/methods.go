package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"io"
	"strings"
	"text/template"
)

var (
	hndlTmpl = template.Must(template.New("hndlTmpl").Parse(`
func (s {{.Recv}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) (d interface{}, e error){
	{{ if .Auth -}}
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	
	{{ end -}}
	
	{{ if .Meth -}}
	if r.Method != "{{ .Meth }}" {
		return nil, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")}
	}
	
	{{ end -}}
	
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := ioutil.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	in, err := New{{.Argt}}(params)
	if err != nil {
		return nil, err
	}

	return s.{{.Name}}(r.Context(), in)
}
	`))

	httpTmpl = template.Must(template.New("httpTmpl").Parse(`
func (s {{.Recv}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		out interface{}
	)

	switch r.URL.Path {
	{{ range .Methods }}case "{{ .Url }}":
		out, err = s.handle{{ .Name }}(w, r)
	{{ end }}default:
		err = ApiError{Err: fmt.Errorf("unknown method"), HTTPStatus: http.StatusNotFound}
	}

	response := struct {
		Data  interface{} ` + "`" + `json:"response,omitempty"` + "`" + `
		Error string      ` + "`" + `json:"error"` + "`" + `
	}{}

	if err == nil {
		response.Data = out
	} else {
		response.Error = err.Error()
		if errApi, ok := err.(ApiError); ok {
			w.WriteHeader(errApi.HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	
	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
	`))
)

type MethodTemplate struct {
	Name string
	Arg  string
	Recv string
	Args string
	Argt string
	Rets string
	Rett string
	Url  string

	Auth bool
	Meth string
}

type HttpTemplate struct {
	Recv    string
	Methods []MethodTemplate
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

	auth bool
	meth string

	child []*MethodHandler // calls
}

func (mh *MethodHandler) Template() MethodTemplate {
	return MethodTemplate{
		Name: mh.n,
		Arg:  mh.arg,
		Recv: mh.rec,
		Args: strings.Join(mh.args, ", "),
		Argt: strings.Join(mh.argt, ", "),
		Rets: strings.Join(mh.rets, ", "),
		Rett: strings.Join(mh.rett, ", "),
		Url:  mh.url,
		Auth: mh.auth,
		Meth: mh.meth,
	}
}

func (mh *MethodHandler) HttpTemplate() HttpTemplate {
	rv := HttpTemplate{}
	rv.Recv = mh.rec

	for _, m := range mh.child {
		rv.Methods = append(rv.Methods, m.Template())
	}

	return rv
}

func (mh *MethodHandler) PrintHandler(w io.Writer) {
	// func declaration
	fmt.Fprintf(w, "func (%s %s) ", mh.arg, mh.rec)
	fmt.Fprintf(w, "handle%s(w http.ResponseWriter, r *http.Request) {\n", mh.n)

	// args definitions
	for i := range mh.argt {
		fmt.Fprintf(w, "\tvar %s %s\n", mh.args[i], mh.argt[i])
	}

	// actuall call
	fmt.Fprintf(w, "\t%s := ", strings.Join(mh.rett, ", "))
	fmt.Fprintf(w, "%s.%s(r.Context(), %s)\n", mh.arg, mh.n, strings.Join(mh.args, ", "))

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
		// rv.args = append(rv.args, r.Names[0].Name)
		switch xv := r.Type.(type) {
		case *ast.StarExpr:
			if si, ok := xv.X.(*ast.Ident); ok {
				rv.argt = append(rv.argt, "*"+si.Name)
				rv.args = append(rv.args, strings.ToLower(si.Name))
			}
		case *ast.Ident:
			rv.argt = append(rv.argt, xv.Name)
			rv.args = append(rv.args, strings.ToLower(xv.Name))
			// case *ast.SelectorExpr:
			// 	if si, ok := xv.X.(*ast.Ident); ok {
			// 		rv.argt = append(rv.argt, si.Name+"."+xv.Sel.Name)
			// 	}
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
	api := struct {
		Meth string `json:"method"`
		Url  string `json:"url"`
		Auth bool   `json:"auth"`
	}{}

	for _, r := range f.Doc.List {
		s := r.Text[len("// apigen:api"):]
		json.Unmarshal([]byte(s), &api)
	}

	if api.Url == "" {
		return nil, fmt.Errorf("url is not specified")
	}

	rv.url = api.Url
	rv.auth = api.Auth
	rv.meth = api.Meth

	return rv, nil
}

func needCodegen(doc *ast.CommentGroup, prefix string) bool {
	nc := false

	if doc == nil {
		return nc
	}

	for _, c := range doc.List {
		nc = nc || strings.HasPrefix(c.Text, prefix)
	}

	return nc
}

func ParseMethods(file *ast.File) ([]*MethodHandler, error) {
	mm := map[string][]MethodHandler{}

	for _, d := range file.Decls {
		f, ok := d.(*ast.FuncDecl)
		if ok {
			if !needCodegen(f.Doc, "// apigen:api") {
				continue
			}
			// only for struct methods
			if f.Recv == nil {
				continue
			}

			mh, e := ParseMethodHandler(f)
			if e != nil {
				return nil, e
			}
			mm[mh.rec] = append(mm[mh.rec], *mh)
		}
	}

	if len(mm) == 0 {
		return nil, fmt.Errorf("no marked methods to generate api")
	}

	mhs := make([]*MethodHandler, 0)

	for _, m := range mm {
		mh := &MethodHandler{
			rec: m[0].rec,
			arg: m[0].arg,
		}
		for i := range m {
			mh.child = append(mh.child, &m[i])
		}
		mhs = append(mhs, mh)
	}

	return mhs, nil
}
