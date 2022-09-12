package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// код писать тут
type TestCase struct {
	ID       string
	IsError  bool
	Expected *ExpectedResult
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
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "error")
}

func TestSearchUser(t *testing.T) {
	ts := NewTestServer("1234")
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			ID: "0",
			Expected: &ExpectedResult{
				age:     "22",
				company: "HOPELI",
				mail:    "boydwolf@hopeli.com",
			},
			IsError: false,
		},
	}

	for caseNum, item := range cases {
		result, err := ts.Search.FindUsers(SearchRequest{})

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Expected, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Expected, result)
		}
	}
}
