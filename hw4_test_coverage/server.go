package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrorMissedField    = fmt.Errorf("field missed")
	ErrorStrIntCast     = fmt.Errorf("couldnt cast s to i")
	ErrorWrongValue     = fmt.Errorf("wrong value")
	ErrorWrongQuery     = fmt.Errorf("query is wrong")
	ErrorInvalidOrder   = fmt.Errorf("ErrorBadOrderField")
	ErrorInvalidOrderby = fmt.Errorf("should be [-1:1]")
)

type root struct {
	XMLName xml.Name `xml:"root"`
	Users   []row    `xml:"row"`
}

type row struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type server struct {
	// access token
	token string
	// users data
	users []User
}

func NewServer(token string, fp string) (s *server, e error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, errors.Wrap(err, "couldnt open file")
	}

	root := &root{}
	err = xml.Unmarshal(data, root)
	if err != nil {
		return nil, errors.Wrap(err, "couldnt parse xml data")
	}

	rv := &server{}
	rv.token = token
	rv.users = make([]User, 0)
	for _, u := range root.Users {
		rv.users = append(rv.users, *u.Convert())
	}

	s = rv

	return s, nil
}

func (r *row) Convert() *User {
	return &User{
		Id:     r.Id,
		Name:   r.FirstName + " " + r.LastName,
		Age:    r.Age,
		Gender: r.Gender,
		About:  r.About,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.token != r.Header.Get("AccessToken") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sc, err := s.SearchRequest(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		j, _ := json.Marshal(SearchErrorResponse{Error: errors.Cause(err).Error()})
		w.Write(j)
		return
	}

	data, code := s.SearchUsers(sc)
	w.WriteHeader(code)
	w.Write(data)
}

func (s *server) SearchRequest(req string) (*SearchRequest, error) {
	q, err := url.ParseQuery(req)
	if err != nil {
		return nil, errors.Wrap(ErrorWrongQuery, "couldnt parse req")
	}

	sr := &SearchRequest{}

	if q.Get("limit") == "" {
		return nil, errors.Wrap(ErrorMissedField, "limit")
	}
	sr.Limit, err = strconv.Atoi(q.Get("limit"))
	if err != nil {
		return nil, errors.Wrap(ErrorStrIntCast, "limit")
	}
	if sr.Limit < 0 {
		return nil, errors.Wrap(ErrorWrongValue, "limit should be positive")
	}

	if q.Get("offset") == "" {
		return nil, errors.Wrap(ErrorMissedField, "field")
	}
	sr.Offset, err = strconv.Atoi(q.Get("offset"))
	if err != nil {
		return nil, errors.Wrap(ErrorStrIntCast, "offset")
	}
	if sr.Offset < 0 {
		return nil, errors.Wrap(ErrorWrongValue, "offset should be positive")
	}
	if sr.Offset > len(s.users) {
		return nil, errors.Wrap(ErrorWrongValue, "offset is too big")
	}

	sr.OrderBy, err = strconv.Atoi(q.Get("order_by"))
	if err != nil {
		return nil, errors.Wrap(ErrorStrIntCast, "order_by")
	}

	if sr.Limit > 25 {
		sr.Limit = 25
	}

	switch sr.OrderBy {
	case OrderByAsIs:
	case OrderByAsc:
	case OrderByDesc:
	default:
		return nil, errors.Wrap(ErrorInvalidOrderby, fmt.Sprint(sr.OrderBy))
	}

	sr.Query = q.Get("query")
	sr.OrderField = q.Get("order_field")

	switch sr.OrderField {
	case "":
		fallthrough
	case "Id":
		fallthrough
	case "Age":
		fallthrough
	case "Name":
	default:
		return nil, errors.Wrap(ErrorInvalidOrder, sr.OrderField)
	}

	return sr, nil
}

func sortbynm(a, b User, inc int) bool {
	if inc > 0 {
		return a.Name < b.Name
	}
	return a.Name > b.Name
}

func sortbyid(a, b User, inc int) bool {
	if inc > 0 {
		return a.Id < b.Id
	}
	return a.Id > b.Id
}

func sortbyag(a, b User, inc int) bool {
	if inc > 0 {
		return a.Age < b.Age
	}
	return a.Age > b.Age
}

func sortUsers(u []User, order string, inc int) []User {
	if inc == OrderByAsIs {
		return u
	}

	var sorter func(User, User, int) bool

	switch order {
	case "Id":
		sorter = sortbyid
	case "Age":
		sorter = sortbyag
	case "":
		fallthrough
	case "Name":
		sorter = sortbynm
	}

	sort.Slice(u, func(i, j int) bool { return sorter(u[i], u[j], inc) })

	return u
}

func (s *server) SearchUsers(sc *SearchRequest) ([]byte, int) {
	rv := make([]User, 0)

	for i := sc.Offset; i < sc.Limit; i++ {
		if !strings.Contains(s.users[i].Name, sc.Query) && !strings.Contains(s.users[i].About, sc.Query) {
			continue
		}
		rv = append(rv, s.users[i])
	}

	rv = sortUsers(rv, sc.OrderField, sc.OrderBy)

	js, _ := json.Marshal(rv)

	return js, http.StatusOK
}
