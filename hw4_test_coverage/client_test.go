package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubServer struct {
	f func() (int, []byte)
}

func stubWithTimeOut() (int, []byte) {
	time.Sleep(time.Second * 1)
	return http.StatusFound, []byte("")
}

func stubNoTimeOut() (int, []byte) {
	return http.StatusFound, []byte("")
}

func stubBrokenJsonErr() (int, []byte) {
	return http.StatusBadRequest, []byte("jSoN")
}

func stubBrokenJsonOk() (int, []byte) {
	return http.StatusOK, []byte("jSoN")
}

func (s *stubServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, d := s.f()
	w.WriteHeader(c)
	w.Write(d)
}

func stubInternalError() (int, []byte) {
	return http.StatusInternalServerError, []byte("")
}

const AccessToken string = "accesstoken"

func TestSearchClient(t *testing.T) {
	ss, e := NewServer(AccessToken, "dataset.xml")
	assert.NotNil(t, ss)
	assert.NoError(t, e)

	s := httptest.NewServer(ss)
	defer s.Close()

	tests := []struct {
		n string
		t string
		u string
		r SearchRequest
		e error
	}{
		{"1", AccessToken, "", SearchRequest{Limit: -1}, fmt.Errorf("limit must be > 0")},
		{"2", AccessToken, "", SearchRequest{Offset: -1}, fmt.Errorf("offset must be > 0")},
		{"3", AccessToken + "?", s.URL, SearchRequest{Query: "any"}, fmt.Errorf("Bad AccessToken")},
		{"4", AccessToken, s.URL, SearchRequest{OrderField: "any"}, fmt.Errorf("OrderFeld %s invalid", "any")},
		{"5", AccessToken, s.URL, SearchRequest{OrderBy: 2}, fmt.Errorf("unknown bad request error: %s", ErrorInvalidOrderby.Error())},
		{"6", AccessToken, s.URL, SearchRequest{Query: "Boyd", Limit: 30}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.n, func(t *testing.T) {
			srv := &SearchClient{
				AccessToken: tt.t,
				URL:         tt.u,
			}
			_, err := srv.FindUsers(tt.r)
			assert.Equal(t, tt.e, err)
		})
	}
}

func TestSearchClientStubs(t *testing.T) {
	ss := &stubServer{}
	s := httptest.NewServer(ss)
	defer s.Close()

	tests := []struct {
		n string
		f func() (int, []byte)
		e error
	}{
		{"1", stubNoTimeOut, fmt.Errorf("unknown error Get \"%s?limit=1&offset=0&order_by=0&order_field=&query=\": 302 response missing Location header", s.URL)},
		{"2", stubWithTimeOut, fmt.Errorf("timeout for limit=1&offset=0&order_by=0&order_field=&query=")},
		{"3", stubBrokenJsonOk, fmt.Errorf("cant unpack result json: invalid character 'j' looking for beginning of value")},
		{"4", stubInternalError, fmt.Errorf("SearchServer fatal error")},
		{"5", stubBrokenJsonErr, fmt.Errorf("cant unpack error json: invalid character 'j' looking for beginning of value")},
	}
	for _, tt := range tests {
		ss.f = tt.f
		t.Run(tt.n, func(t *testing.T) {
			srv := &SearchClient{
				AccessToken: AccessToken,
				URL:         s.URL,
			}
			_, err := srv.FindUsers(SearchRequest{})
			assert.Equal(t, tt.e, err)
		})
	}
}

func TestSearchClientUsers(t *testing.T) {
	ss, e := NewServer(AccessToken, "dataset.xml")
	assert.NotNil(t, ss)
	assert.NoError(t, e)

	s := httptest.NewServer(ss)
	defer s.Close()

	tests := []struct {
		n string
		r SearchRequest
		u []User
	}{
		{"1", SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[1], ss.users[2], ss.users[3]}},
		{"2", SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[1]}},
		{"3", SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByDesc}, []User{ss.users[1], ss.users[0], ss.users[2], ss.users[3]}},
		{"4", SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[0]}},
		{"5", SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[2], ss.users[3], ss.users[1]}},
		{"6", SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}},
		{"7", SearchRequest{Limit: 4, OrderField: "", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}},
	}
	for _, tt := range tests {
		t.Run(tt.n, func(t *testing.T) {
			srv := &SearchClient{
				AccessToken: AccessToken,
				URL:         s.URL,
			}
			r, err := srv.FindUsers(tt.r)
			assert.Nil(t, err)
			assert.Equal(t, tt.u, r.Users)
		})
	}
}
