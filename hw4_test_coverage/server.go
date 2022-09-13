package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"google.golang.org/protobuf/internal/errors"
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

	for _, u := range root.Users {
		s.users = append(s.users, *u.Convert())
	}
	s.token = token

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

func sendError() {

}

func (s *server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if s.token != r.Header.Get("AccessToken") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sc, err := s.SearchRequest(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, code := s.SearchUsers(sc)
	w.WriteHeader(code)
	w.Write(data)
}

func (s *server) SearchRequest(req string) (*SearchRequest, error) {
	q, err := url.ParseQuery(req)
	if err != nil {
		return nil, errors.Wrap(err, "couldnt parse req")
	}
	sr := &SearchRequest{}
	sr.Limit, err = strconv.Atoi(q.Get("limit"))
	if err != nil {
		return nil, errors.Wrap(err, "couldnt get limit field")
	}

	sr.Offset, err = strconv.Atoi(q.Get("offset"))
	if err != nil {
		return nil, errors.Wrap(err, "couldnt get offset field")
	}

	sr.Query = q.Get("query")

	return sr, nil
}

func (s *server) SearchUsers(sc *SearchRequest) ([]byte, int) {
	rv := make([]User, 0)

	for i := 0; i < sc.Limit; i++ {
		if !strings.Contains(s.users[i].Name, sc.Query) && !strings.Contains(s.users[i].About, sc.Query) {
			continue
		}
		if i > sc.Offset {
			rv = append(rv, s.users[i])
		}
	}

	js, err := json.Marshal(rv)
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	// if query != "" && len(users) == 0 {
	// 	js, _ := json.Marshal(SearchErrorResponse{"no records with value " + query})
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	w.Header().Set("Content-Type", "application/json")
	// 	io.WriteString(w, string(js))
	// 	return
	// }

	// order_by, _ := strconv.Atoi(q.Get("order_by"))
	// var sorter func(a, b User) bool
	// if order_by != 0 {
	// 	switch q.Get("order_field") {
	// 	case "Id":
	// 		sorter = func(a, b User) bool { return a.Id < b.Id }
	// 	case "Age":
	// 		sorter = func(a, b User) bool { return a.Age < b.Age }
	// 	case "":
	// 	case "Name":
	// 		sorter = func(a, b User) bool { return a.Name < b.Name }
	// 	default:
	// 		js, _ := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
	// 		w.WriteHeader(http.StatusBadRequest)
	// 		w.Header().Set("Content-Type", "application/json")
	// 		io.WriteString(w, string(js))
	// 		return
	// 	}
	// 	sort.Slice(users, func(i, j int) bool {
	// 		return sorter(users[i], users[j]) && (order_by == -1)
	// 	})
	// }

	return js, http.StatusOK
}
