package main

import (
	"go/ast"
	"strconv"
	"strings"
)

const (
	fieldReq = "required"  // поле не должно быть пустым (не должно иметь значение по-умолчанию)
	fieldNam = "paramname" // если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
	fieldEnm = "enum"      // "одно из"
	fieldDef = "default"   // если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
	fieldMin = "min"       // >= X для типа `int`, для строк `len(str)` >=
	fieldMax = "max"       // <= X для типа `int`
)

type FieldParams struct {
	req bool
	def bool
	pnm string
	enm string
	max int
	min int
}

type StructField struct {
	fn string
	fp FieldParams
}

type StructDef struct {
	sn string
	sf []*StructField
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
				sf.fp.req = true
			}
			if strings.HasPrefix(t, fieldDef) {
				sf.fp.def = true
			}
			if strings.HasPrefix(t, fieldMin) {
				v := strings.Split(t, "=")[1]
				sf.fp.min, err = strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(t, fieldMax) {
				v := strings.Split(t, "=")[1]
				sf.fp.min, err = strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(t, fieldEnm) {
				sf.fp.enm = strings.Split(t, "=")[1]
			}
			if strings.HasPrefix(t, fieldNam) {
				sf.fp.pnm = strings.Split(t, "=")[1]
			}
		}
		sf.fn = tag.Names[0].Name
		sfs = append(sfs, sf)
	}

	return sfs, nil
}

func ParseStructs(file *ast.File, names string) ([]*StructDef, error) {
	// st := []*StructDef{}
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

			_, e := parseStructFields(ss, "")
			if e != nil {
				return nil, e
			}
		}
	}

	return nil, nil
}
