package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

// Columns definition
type ColumnType interface {
	Type() interface{}
	Valid(val interface{}) bool
}

type ColumnString struct {
	Null bool
}

func (col *ColumnString) Type() interface{} {
	if col.Null {
		return new(*string)
	}
	return new(string)
}

func (col *ColumnString) Valid(val interface{}) bool {
	if val == nil {
		return col.Null
	}
	_, ok := val.(string)
	return ok
}

type ColumnInteger struct {
	Null bool
}

func (col *ColumnInteger) Type() interface{} {
	if col.Null {
		return new(*int)
	}
	return new(int)
}

func (col *ColumnInteger) Valid(val interface{}) bool {
	if val == nil {
		return col.Null
	}
	_, ok := val.(int)
	return ok
}

type Column struct {
	Field      string
	Type       ColumnType
	Collation  interface{}
	Null       interface{}
	Key        string
	Default    interface{}
	Extra      string
	Privileges string
	Comment    string
}

// Table definitions
type Table struct {
	n string
	k string
	c []Column
}

func (t *Table) newRow() []interface{} {
	row := make([]interface{}, 0, len(t.c))
	for _, v := range t.c {
		row = append(row, v.Type.Type())
	}

	return row
}

type DbExplorer struct {
	db *sql.DB

	tables map[string]Table
}

func (e *DbExplorer) getColumns(table string) ([]Column, error) {
	rows, err := e.db.Query("SHOW FULL COLUMNS FROM " + table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		colType string
		colNull string
		isNull  bool
	)

	rv := make([]Column, 0)
	col := Column{}

	for rows.Next() {
		if err := rows.Scan(
			&col.Field,
			&colType,
			&col.Collation,
			&colNull,
			&col.Key,
			&col.Default,
			&col.Extra,
			&col.Privileges,
			&col.Comment,
		); err != nil {
			return nil, err
		}

		isNull = colNull == "YES"
		if strings.Contains(colType, "int") {
			col.Type = &ColumnInteger{Null: isNull}
		} else {
			col.Type = &ColumnString{Null: isNull}
		}

		rv = append(rv, col)
	}

	return rv, nil
}

func (e *DbExplorer) getTablesNames() ([]string, error) {
	row, err := e.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer row.Close()

	s := ""
	rv := make([]string, 0)
	for row.Next() {
		err = row.Scan(&s)
		if err != nil {
			return nil, err
		}

		rv = append(rv, s)
	}

	return rv, nil
}

func (e *DbExplorer) getTables() (map[string]Table, error) {
	names, err := e.getTablesNames()
	if err != nil {
		return nil, err
	}

	rv := make(map[string]Table, len(names))
	for _, name := range names {
		cols, err := e.getColumns(name)
		if err != nil {
			return nil, err
		}
		t := Table{
			c: cols,
			n: name,
		}

		for _, col := range cols {
			if col.Key == "PRI" {
				t.k = col.Field
				break
			}
		}

		rv[name] = t
	}

	return rv, nil
}

func (h *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Data  interface{} `json:"response,omitempty"`
		Error string      `json:"error,omitempty"`
	}{}

	status := http.StatusOK

	rv, err := h.handlerRouter(r)
	if rv == nil {
		res.Error = err.(*DbResponseError).e
		status = err.(*DbResponseError).s
	} else {
		res.Data = rv
	}

	js, _ := json.MarshalIndent(res, "", " ")
	w.WriteHeader(status)
	w.Write(js)
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	rv := &DbExplorer{
		db: db,
	}

	t, e := rv.getTables()
	if e != nil {
		return nil, e
	}

	rv.tables = t

	return rv, nil
}
