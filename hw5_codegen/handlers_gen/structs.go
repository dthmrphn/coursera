package main

import (
	"go/ast"
	"strconv"
	"strings"
	"text/template"
)

const (
	fieldReq = "required"  // поле не должно быть пустым (не должно иметь значение по-умолчанию)
	fieldNam = "paramname" // если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
	fieldEnm = "enum"      // "одно из"
	fieldDef = "default"   // если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
	fieldMin = "min"       // >= X для типа `int`, для строк `len(str)` >=
	fieldMax = "max"       // <= X для типа `int`
)

var (
	structTmpl = template.Must(template.New("structTmpl").Parse(`
func New{{.Name}}(p url.Values) ({{.Name}}, error) {
	s := {{.Name}}{}
	var err error = nil

	{{ range .Fields}} //{{ .Name }}

	{{- if eq .Type "int" }}
	s.{{ .Name }}, err = strconv.Atoi(p.Get("{{ .Params.ParamName }}"))
	if err != nil {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} must be int")}
	}
	{{ else }}
	s.{{ .Name }} = p.Get("{{ .Params.ParamName }}")

	{{ end -}}

	{{- if .Params.Default -}}
	if s.{{ .Name }} == "" {
		s.{{ .Name }} = "{{ .Params.Default }}"
	}

	{{ end -}}

	{{- if .Params.Required -}}
	if s.{{ .Name }} == "" {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} must me not empty")}
	}

	{{ end -}}

	{{- if and .Params.Min (eq .Type "int") -}}
	if s.{{ .Name }} < {{ .Params.MinValue }} {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} must be >= {{ .Params.MinValue }}")}
	}

	{{ end -}}

	{{- if and .Params.Max (eq .Type "int") -}}
	if s.{{ .Name }} > {{ .Params.MaxValue }} {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} must be <= {{ .Params.MaxValue }}")}
	}

	{{ end -}}

	{{- if and .Params.Min (eq .Type "string") -}}
	if len(s.{{ .Name }}) < {{ .Params.MinValue }} {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} len must be >= {{ .Params.MinValue }}")}
	}

	{{ end -}}

	{{- if and .Params.Max (eq .Type "string") -}}
	if len(s.{{ .Name }}) > {{ .Params.MaxValue }} {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} len must be <= {{ .Params.MaxValue }}")}
	}

	{{ end -}}

	{{- if .Params.Enum -}}
	enum{{ .Name }}Valid := false
	enum{{ .Name }} := []string{ {{- range $index, $element := .Params.Enums }}{{ if $index }}, {{ end }}"{{ $element }}"{{ end -}} }
	for _, valid := range enum{{ .Name }} {
		if valid == s.{{ .Name }} {
			enum{{ .Name }}Valid = true
			break
		}
	}
	if !enum{{ .Name }}Valid {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("{{ .Params.ParamName }} must be one of [%s]", strings.Join(enum{{ .Name }}, ", "))}
	}

	{{ end -}}

	{{- end -}}

	return s, err
}
	`))
)

type FieldParams struct {
	Required  bool
	Default   string
	ParamName string
	Enum      bool
	Enums     []string
	MaxValue  int
	MinValue  int
	Max       bool
	Min       bool
}

type StructField struct {
	Name   string
	Type   string
	Params FieldParams
}

type StructDef struct {
	Name   string
	Fields []*StructField
}

type StructTemplate struct {
	Name   string
	Fields []*StructField
}

func (s *StructDef) Template() StructTemplate {
	return StructTemplate{
		Name:   s.Name,
		Fields: s.Fields,
	}
}

func parseStructFields(s *ast.StructType, fn string) ([]*StructField, error) {
	sfs := make([]*StructField, 0)
	for _, tag := range s.Fields.List {
		if tag.Tag == nil {
			continue
		}
		if !strings.HasPrefix(tag.Tag.Value, "`apivalidator:") {
			continue
		}

		sf := &StructField{}
		var err error
		tags := strings.Split(tag.Tag.Value, ":\"")[1]
		for _, t := range strings.Split(tags, ",") {
			t = strings.Trim(t, "\",`")
			if strings.HasPrefix(t, fieldReq) {
				sf.Params.Required = true
			}
			if strings.HasPrefix(t, fieldDef) {
				sf.Params.Default = strings.Split(t, "=")[1]
			}
			if strings.HasPrefix(t, fieldMin) {
				v := strings.Split(t, "=")[1]
				sf.Params.Min = true
				sf.Params.MinValue, err = strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(t, fieldMax) {
				v := strings.Split(t, "=")[1]
				sf.Params.Max = true
				sf.Params.MaxValue, err = strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(t, fieldEnm) {
				sf.Params.Enum = true
				v := strings.Split(t, "=")[1]
				sf.Params.Enums = strings.Split(v, "|")
			}
			if strings.HasPrefix(t, fieldNam) {
				sf.Params.ParamName = strings.Split(t, "=")[1]
			} else {
				sf.Params.ParamName = strings.ToLower(tag.Names[0].Name)
			}
		}
		sf.Type = tag.Type.(*ast.Ident).Name
		sf.Name = tag.Names[0].Name
		sfs = append(sfs, sf)
	}

	return sfs, nil
}

func ParseStructs(file *ast.File, names string, prefix string) ([]*StructDef, error) {
	structs := []*StructDef{}
	for _, d := range file.Decls {
		g, ok := d.(*ast.GenDecl)
		if ok {
			s := g.Specs[0]
			ts, ok := s.(*ast.TypeSpec)
			if !ok {
				continue
			}
			ss, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if !strings.Contains(names, ts.Name.Name) {
				continue
			}

			sf, e := parseStructFields(ss, ts.Name.Name)
			if e != nil {
				return nil, e
			}
			st := &StructDef{
				Name:   ts.Name.Name,
				Fields: sf,
			}
			structs = append(structs, st)
		}
	}

	return structs, nil
}
