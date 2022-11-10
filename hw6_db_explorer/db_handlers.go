package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type DbResponse map[string]interface{}

type DbResponseError struct {
	s int
	e string
}

func (e *DbResponseError) Error() string {
	return e.e
}

func (e *DbExplorer) newDbRequest(r *http.Request) (Table, int, error) {
	id := -1
	table := Table{}
	paths := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(paths) >= 1 {
		v, ok := e.tables[paths[0]]
		if !ok {
			return Table{}, 0, &DbResponseError{http.StatusNotFound, "unknown table"}
		}
		table = v
	}

	if len(paths) >= 2 {
		v, err := strconv.Atoi(paths[1])
		if err != nil {
			return Table{}, 0, &DbResponseError{http.StatusBadRequest, err.Error()}
		}
		id = v
	}

	return table, id, nil
}

func getLimitOffset(r *http.Request) (limit int, offset int, err error) {
	limit = 5
	offset = 0

	l := r.URL.Query().Get("limit")
	if l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			// return
			limit = 5
		}
	}

	o := r.URL.Query().Get("offset")
	if o != "" {
		offset, err = strconv.Atoi(o)
		if err != nil {
			// return
			offset = 0
		}
	}

	err = nil
	return
}

func getPostData(r *http.Request, t Table) (cols []string, vals []interface{}, err error) {
	values := make(map[string]interface{})
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &values)
	if err != nil {
		return
	}

	vals = make([]interface{}, 0, len(values))
	cols = make([]string, 0, len(values))
	for _, c := range t.c {
		if v, ok := values[c.Field]; ok {
			if c.Field == t.k {
				err = fmt.Errorf("field %s have invalid type", c.Field)
				return
			}
			if !c.Type.Valid(v) {
				err = fmt.Errorf("field %s have invalid type", c.Field)
				return
			}
			vals = append(vals, v)
			cols = append(cols, fmt.Sprintf("%s = ?", c.Field))
		}
	}

	return
}

func getPutData(r *http.Request, t Table) (cols []string, vals []interface{}, err error) {
	values := make(map[string]interface{})
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &values)
	if err != nil {
		return
	}

	vals = make([]interface{}, 0, len(values))
	cols = make([]string, 0, len(values))
	for _, c := range t.c {
		if c.Field == t.k {
			continue
		}

		cols = append(cols, c.Field)

		if v, ok := values[c.Field]; ok {
			if !c.Type.Valid(v) {
				err = fmt.Errorf("field %s have invalid type", c.Field)
				return
			}
			vals = append(vals, v)
		} else {
			vals = append(vals, c.Type.Type())
		}
	}

	return
}

func (e *DbExplorer) handlerMethodGet(r *http.Request) (DbResponse, error) {
	rv := make(map[string]interface{})

	if r.URL.Path == "/" {
		tables := make([]string, 0, len(e.tables))
		for table, _ := range e.tables {
			tables = append(tables, table)
		}

		rv["tables"] = tables
		return rv, &DbResponseError{http.StatusOK, ""}
	}

	table, id, err := e.newDbRequest(r)
	if err != nil {
		return nil, err
	}

	limit, offset, err := getLimitOffset(r)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	q := ""
	if id < 0 {
		q = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", table.n, limit, offset)
	} else {
		q = fmt.Sprintf("SELECT * FROM %s WHERE %s = %d", table.n, table.k, id)
	}

	rows, err := e.db.Query(q)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}
	defer rows.Close()

	records := make([]interface{}, 0)
	for rows.Next() {
		r := table.newRow()
		rows.Scan(r...)
		record := map[string]interface{}{}
		for i, c := range table.c {
			record[c.Field] = r[i]
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, &DbResponseError{http.StatusNotFound, "record not found"}
	}

	if id < 0 {
		rv["records"] = records
	} else {
		rv["record"] = records[0]
	}

	return rv, nil
}

func (e *DbExplorer) handlerMethodPut(r *http.Request) (DbResponse, error) {
	rv := make(map[string]interface{})

	table, _, err := e.newDbRequest(r)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	cols, vals, err := getPutData(r, table)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table.n,
		strings.Join(cols, ", "),
		strings.Join(strings.Split(strings.Repeat("?", len(cols)), ""), ", "),
	)

	res, err := e.db.Exec(q, vals...)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	rv[table.k] = id

	return rv, nil
}

func (e *DbExplorer) handlerMethodPost(r *http.Request) (DbResponse, error) {
	rv := make(map[string]interface{})

	table, id, err := e.newDbRequest(r)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	cols, vals, err := getPostData(r, table)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}
	vals = append(vals, id)

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		table.n,
		strings.Join(cols, ", "),
		table.k,
	)

	res, err := e.db.Exec(q, vals...)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	upd, err := res.RowsAffected()
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	rv["updated"] = upd

	return rv, nil
}

func (e *DbExplorer) handlerMethodDelete(r *http.Request) (DbResponse, error) {
	rv := make(map[string]interface{})

	table, id, err := e.newDbRequest(r)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ?;",
		table.n,
		table.k,
	)

	res, err := e.db.Exec(q, id)
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	del, err := res.RowsAffected()
	if err != nil {
		return nil, &DbResponseError{http.StatusBadRequest, err.Error()}
	}

	rv["deleted"] = del

	return rv, nil
}

func (e *DbExplorer) handlerRouter(r *http.Request) (DbResponse, error) {
	switch r.Method {
	case http.MethodGet:
		return e.handlerMethodGet(r)
	case http.MethodPut:
		return e.handlerMethodPut(r)
	case http.MethodPost:
		return e.handlerMethodPost(r)
	case http.MethodDelete:
		return e.handlerMethodDelete(r)
	default:
		return nil, &DbResponseError{http.StatusBadRequest, "unknown method"}
	}
}
