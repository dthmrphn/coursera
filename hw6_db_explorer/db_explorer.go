package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type DbResponseError struct {
	e string
	s int
}

func (e *DbResponseError) Error() string {
	return e.e
}

type DbResponse struct {
	r  map[string]interface{}
	re *DbResponseError
}

type Column struct {
	n string
	t string
	v interface{}
}

type Table struct {
	n string
	k string
	c []Column
}

func (t *Table) newRow() []string {
	row := make([]string, len(t.c))
	for i := range row {
		row[i] = ""
	}

	return row
}

type DbRequest struct {
	table Table
	id    int
}

type DbExplorer struct {
	db *sql.DB

	tables map[string]Table
}

func (e *DbExplorer) getColumns(table string) ([]Column, error) {
	row, err := e.db.Query(fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer row.Close()

	cols, err := row.Columns()
	if err != nil {
		return nil, err
	}

	rv := make([]Column, 0, len(cols))
	for _, c := range cols {
		col := Column{
			n: c,
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
		c, err := e.getColumns(name)
		if err != nil {
			return nil, err
		}
		t := Table{
			c: c,
			n: name,
		}
		rv[name] = t
	}

	return rv, nil
}

func (e *DbExplorer) newRequest(r *http.Request) (*DbRequest, error) {
	if r.URL.Path == "/" {
		return nil, nil
	}

	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	rv := &DbRequest{}

	if len(paths) >= 1 {
		if _, ok := e.tables[paths[0]]; !ok {
			return nil, &DbResponseError{"unknown table", http.StatusNotFound}
		}
		rv.table = e.tables[paths[0]]
	}

	if len(paths) >= 2 {
		if id, err := strconv.Atoi(paths[1]); err == nil {
			rv.id = id
		}
	}

	return rv, nil
}

func (e *DbExplorer) handlerGetMethod(req *DbRequest) DbResponse {
	rv := DbResponse{
		r:  map[string]interface{}{},
		re: &DbResponseError{"", http.StatusOK},
	}

	if req == nil {
		tables := make([]string, 0, len(e.tables))
		for table, _ := range e.tables {
			tables = append(tables, table)
		}

		rv.r["tables"] = tables
		return rv
	}

	// limit := 5
	// offset := 0
	// q := fmt.Sprintf("SELECT * FROM %s  LIMIT ? OFFSET ?", req.table.n)
	// rows, err := e.db.Query(q, limit, offset)
	q := fmt.Sprintf("SELECT * FROM %s", req.table.n)
	rows, err := e.db.Query(q)
	if err != nil {
		rv.re.e = err.Error()
		return rv
	}
	defer rows.Close()

	// records := make([]map[string]interface{}, 0)

	for rows.Next() {
		row := req.table.newRow()
		if err := rows.Scan(row); err == nil {
			// records = append(records, row)
			fmt.Println(row)
		}
		fmt.Println(row)
	}
	return rv
}

func (e *DbExplorer) RouteQuery(r *http.Request) DbResponse {
	rv := DbResponse{
		r:  map[string]interface{}{},
		re: &DbResponseError{"", http.StatusOK},
	}

	req, err := e.newRequest(r)
	if err != nil {
		rv.re = err.(*DbResponseError)
		return rv
	}

	switch r.Method {
	case http.MethodGet:
		rv = e.handlerGetMethod(req)
	default:
		rv.re = &DbResponseError{"unknown method", http.StatusBadRequest}
	}

	return rv
}

func (h *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Data  interface{} `json:"response,omitempty"`
		Error string      `json:"error,omitempty"`
	}{}

	dbr := h.RouteQuery(r)

	if dbr.re.e != "" {
		res.Error = dbr.re.e
	}

	if dbr.r != nil {
		res.Data = dbr.r
	}

	js, _ := json.Marshal(res)

	w.WriteHeader(dbr.re.s)
	// w.Header().Set("Content-Type", "application/json")
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
