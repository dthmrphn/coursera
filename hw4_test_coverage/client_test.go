package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var content Content

const AccessToken string = "accesstoken"

func init() {
	data, err := os.ReadFile("dataset.xml")
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal(data, &content)
	if err != nil {
		panic(err)
	}
}

type Content struct {
	XMLName xml.Name `xml:"root"`
	Users   []Row    `xml:"row"`
}

type Row struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type TestCase struct {
	sr  *SearchRequest
	exp *Row
}

type TestServer struct {
	Server *httptest.Server
	Search SearchClient
}

func (ts *TestServer) Close() {
	ts.Server.Close()
}

func NewTestServer(token string, URL string) TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		token,
		server.URL + URL,
	}

	return TestServer{
		server,
		client,
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("AccessToken") {
	case "json":
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "{}]")
		return
	case "internal":
		w.WriteHeader(http.StatusInternalServerError)
		return
	case "brokenaccesstoken":
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.URL.Path == "/timeout" {
		w.WriteHeader(http.StatusFound)
		return
	}

	users := make([]User, 0)

	q := r.URL.Query()
	query := q.Get("query")
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit > len(content.Users) {
		limit = len(content.Users)
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	for i, user := range content.Users {
		if query != "" {
			if !strings.Contains(user.FirstName, query) && !strings.Contains(user.LastName, query) && !strings.Contains(user.About, query) {
				i--
				continue
			}
			if i > offset-1 {
				user := user.Convert()
				users = append(users, *user)
			}
			if i > limit {
				break
			}
		} else {
			user := user.Convert()
			users = append(users, *user)
		}
	}

	order_by, _ := strconv.Atoi(q.Get("order_by"))
	var sorter func(a, b User) bool
	if order_by != 0 {
		switch q.Get("order_field") {
		case "Id":
			sorter = func(a, b User) bool { return a.Id < b.Id }
		case "Age":
			sorter = func(a, b User) bool { return a.Age < b.Age }
		case "":
		case "Name":
			sorter = func(a, b User) bool { return a.Name < b.Name }
		default:
			js, _ := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, string(js))
			return
		}
		sort.Slice(users, func(i, j int) bool {
			return sorter(users[i], users[j]) && (order_by == -1)
		})
	}

	js, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (u *User) Marshal() ([]byte, error) {
	return json.MarshalIndent(u, "", "\t")
}

func (r *Row) Convert() *User {
	return &User{
		Id:     r.Id,
		Name:   r.FirstName + " " + r.LastName,
		Age:    r.Age,
		Gender: r.Gender,
		About:  r.About,
	}
}

func TestSearchUserQuery(t *testing.T) {
	ts := NewTestServer(AccessToken, "")
	defer ts.Close()

	sc_right := SearchRequest{
		Limit:      1,
		Query:      "Boyd",
		OrderBy:    -1,
		OrderField: "Name",
	}

	sc_wrong := SearchRequest{
		Limit:      1,
		Query:      "Boyd",
		OrderBy:    -1,
		OrderField: "WRONG",
	}

	usr_right := &Row{
		Age:       22,
		Gender:    "male",
		FirstName: "Boyd",
		LastName:  "Wolf",
	}

	usr_wrong := &Row{
		Age:    22,
		Gender: "male",
	}

	tc := []TestCase{
		{&sc_right, usr_right},
		{&sc_wrong, usr_wrong},
	}

	result, err := ts.Search.FindUsers(*tc[0].sr)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Users[0].Age, tc[0].exp.Age)
	assert.Equal(t, result.Users[0].Gender, tc[0].exp.Gender)
	assert.Equal(t, result.Users[0].Name, tc[0].exp.FirstName+" "+tc[0].exp.LastName)

	_, err = ts.Search.FindUsers(*tc[1].sr)
	assert.NotEqual(t, err, nil)
}

func TestSearchUserOrder(t *testing.T) {
	ts := NewTestServer(AccessToken, "")
	defer ts.Close()

	sr_inc := &SearchRequest{
		Limit:      2,
		Query:      "Boyd",
		OrderBy:    1,
		OrderField: "Age",
	}

	sr_dec := &SearchRequest{
		Limit:      2,
		Query:      "Boyd",
		OrderBy:    -1,
		OrderField: "Age",
	}

	sr_err := &SearchRequest{
		Limit:      2,
		Query:      "Boyd",
		OrderBy:    -1,
		OrderField: "wrong",
	}

	usr1 := &Row{
		Age: 22,
	}

	usr2 := &Row{
		Age: 21,
	}

	result, err := ts.Search.FindUsers(*sr_inc)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Users[0].Age, usr1.Age)
	assert.Equal(t, result.Users[1].Age, usr2.Age)

	result, err = ts.Search.FindUsers(*sr_dec)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Users[1].Age, usr1.Age)
	assert.Equal(t, result.Users[0].Age, usr2.Age)

	_, err = ts.Search.FindUsers(*sr_err)
	assert.NotEqual(t, err, nil)
}

func TestSearchUserOrderLimits(t *testing.T) {
	ts := NewTestServer(AccessToken, "")
	defer ts.Close()

	sr1 := &SearchRequest{
		Offset:     0,
		Limit:      1,
		Query:      "Boyd",
		OrderField: "Age",
	}

	sr2 := &SearchRequest{
		Offset:     0,
		Limit:      2,
		Query:      "Boyd",
		OrderField: "Age",
	}

	usr1 := &Row{
		Age: 22,
	}

	usr2 := &Row{
		Age: 21,
	}

	result, err := ts.Search.FindUsers(*sr1)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Users[0].Age, usr1.Age)

	result, err = ts.Search.FindUsers(*sr2)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Users[0].Age, usr1.Age)
	assert.Equal(t, result.Users[1].Age, usr2.Age)
}

func TestLimits(t *testing.T) {
	sc := SearchClient{}
	_, err := sc.FindUsers(SearchRequest{
		Limit: -1,
	})
	assert.Error(t, err)

	_, err = sc.FindUsers(SearchRequest{
		Limit:  100,
		Offset: -1,
	})
	assert.Error(t, err)
}

func TestWrongToken(t *testing.T) {
	ts := NewTestServer("broken"+AccessToken, "")
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{Query: "any"})

	assert.NotEqual(t, err, nil)
}

func TestInternal(t *testing.T) {
	ts := NewTestServer("internal", "")
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{Query: "any"})

	assert.NotEqual(t, err, nil)
}

func TestWrongJSON(t *testing.T) {
	ts := NewTestServer("json", "")
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{Query: "any"})

	assert.NotEqual(t, err, nil)
}

func TestTimeout(t *testing.T) {
	ts := NewTestServer(AccessToken, "/timeout")
	defer ts.Close()

	_, err := ts.Search.FindUsers(SearchRequest{Query: "any"})

	assert.NotEqual(t, err, nil)
}

func TestUniqNames(t *testing.T) {
	m := map[string]int{}

	for _, u := range content.Users {
		m[u.About]++

	}
	fmt.Print(m)
}
