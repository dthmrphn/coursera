package main

import (
	"encoding/json"
	"encoding/xml"
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
	Id     int    `xml:"id"`
	Name   string `xml:"first_name"`
	Age    int    `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
}

type TestCase struct {
	sr *SearchRequest
	exp *Row
}

type ExpectedResult struct {
	age     string
	company string
	mail    string
}

type TestServer struct {
	Server *httptest.Server
	Search SearchClient
}

func (ts *TestServer) Close() {
	ts.Server.Close()
}

func NewTestServer(token string) TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		token,
		server.URL,
	}

	return TestServer{
		server,
		client,
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	
	users := make([]User, 0)

	q := r.URL.Query();
	query := q.Get("query")
	for _, user := range content.Users {
		if query != "" {
			if !strings.Contains(user.Name, query) && !strings.Contains(user.About, query) {
				continue
			}
			user := user.Convert()
			users = append(users, *user)
		}
	}
	
	order_by, _:= strconv.Atoi(q.Get("order_by"))
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
			// error
		}
		sort.Slice(users, func (i, j int) bool {
			return sorter(users[i], users[j]) && (order_by == -1)
		})
	} 

	// limit, _ := strconv.Atoi(r.FormValue("limit"))
	// offset, _ := strconv.Atoi(r.FormValue("offset"))

	// if limit > 25 {
	// 	limit = 25
	// }

	// if offset + limit > len(content.Users) {
	// 	offset = len(content.Users)
	// }

	js, _ := json.Marshal(users)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (u *User) Marshal() ([]byte, error){
	return json.MarshalIndent(u, "", "\t")
}

func (r *Row) Convert() *User {
	return &User {
		Id: r.Id,
		Name: r.Name,
		Age: r.Age,
		Gender: r.Gender,
		About: r.About,
	}
}

func TestSearchUserQuery(t *testing.T) {
	ts := NewTestServer("1234")
	defer ts.Close()

	// cases := []TestCase{
	// 	TestCase{
	// 		ID: "0",
	// 		Expected: &Row{
	// 			Age:     22,
	// 			Gender: "male",
	// 		},
	// 		IsError: false,
	// 	},
	// }

	// for _, item := range cases {
	// 	result, _ := ts.Search.FindUsers(SearchRequest{
	// 		Limit: 1,
	// 		Query: "Boyd",
	// 	})

	// 	assert.Equal(t, result.Users[0].Age, item.Expected.Age)
	// 	assert.Equal(t, result.Users[0].Gender, item.Expected.Gender)
	// }
}

func TestSearchUserOrder(t *testing.T) {
	ts := NewTestServer("1234")
	defer ts.Close()

	tc := []TestCase {
		TestCase{
			
		}
	}

	sc_right := SearchRequest {
		Limit: 1,
		Query: "Boyd",
		OrderBy: -1,
		OrderField: "Name",
	}

	sc_wrong := SearchRequest {
		Limit: 1,
		Query: "Boyd",
		OrderBy: -1,
		OrderField: "WRONG",
	}

	tc_right := TestCase {
			ID: "0",
			Expected: &Row{
				Age:     22,
				Gender: "male",
			},
			IsError: false,
	}

	tc_wrong := TestCase {
		ID: "77",
		Expected: &Row{
			Age:     22,
			Gender: "male",
		},
		IsError: false,
	}

	for _, item := range cases {
		result, _ := ts.Search.FindUsers(SearchRequest{
			Limit: 1,
			Query: "Boyd",
			OrderField: "Name",
			OrderBy: -1,
		})

		assert.Equal(t, result.Users[0].Age, item.Expected.Age)
		assert.Equal(t, result.Users[0].Gender, item.Expected.Gender)
	}
}

func TestXML(t *testing.T) {
	if content.Users[0].Name != "Boyd" {
		t.Errorf("expected: Boyd, got : %s", content.Users[0].Name)
	}

	if content.Users[34].Name != "Kane" {
		t.Errorf("expected: Kane, got : %s", content.Users[34].Name)
	}
}

func TestUser(t *testing.T) {
	user := content.Users[0]

	data, _ := user.Convert().Marshal()
	
	assert.Equal(t, string(data), "")
}

func TestLimits(t *testing.T) {
	sc := SearchClient{}

	_, err := sc.FindUsers(SearchRequest{
		Limit: -1,
	})
	assert.Error(t, err)

	_, err = sc.FindUsers(SearchRequest{
		Limit: 100,
		Offset: -1,
	})
	assert.Error(t, err)
}
