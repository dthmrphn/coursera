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

	tests := []struct {
		n string
		t string
		u string
		r SearchRequest
		e error
	}{
		{"1", AccessToken, "", SearchRequest{Limit: -1}, fmt.Errorf("limit must be > 0")},
		{"2", AccessToken, "", SearchRequest{Offset: -1}, fmt.Errorf("offset must be > 0")},
		{"3", "AcEstOk3n", s.URL, SearchRequest{Query: "any"}, fmt.Errorf("Bad AccessToken")},
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
	s_jo := httptest.NewServer(&stubServer{f: stubBrokenJsonOk})
	s_je := httptest.NewServer(&stubServer{f: stubBrokenJsonErr})
	s_tw := httptest.NewServer(&stubServer{f: stubWithTimeOut})
	s_tn := httptest.NewServer(&stubServer{f: stubNoTimeOut})
	s_ie := httptest.NewServer(&stubServer{f: stubInternalError})

	tests := []struct {
		n string
		t string
		u string
		r SearchRequest
		e error
	}{
		{"1", AccessToken, s_jo.URL, SearchRequest{}, fmt.Errorf("cant unpack result json: invalid character 'j' looking for beginning of value")},
		{"2", AccessToken, s_je.URL, SearchRequest{}, fmt.Errorf("cant unpack error json: invalid character 'j' looking for beginning of value")},
		{"3", AccessToken, s_tw.URL, SearchRequest{}, fmt.Errorf("timeout for limit=1&offset=0&order_by=0&order_field=&query=")},
		{"4", AccessToken, s_tn.URL, SearchRequest{}, fmt.Errorf("unknown error Get \"%s?limit=1&offset=0&order_by=0&order_field=&query=\": 302 response missing Location header", s_tn.URL)},
		{"5", AccessToken, s_ie.URL, SearchRequest{}, fmt.Errorf("SearchServer fatal error")},
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

func TestSearchClientUsers(t *testing.T) {
	ss, e := NewServer(AccessToken, "dataset.xml")
	assert.NotNil(t, ss)
	assert.NoError(t, e)

	s := httptest.NewServer(ss)

	tests := []struct {
		n string
		r SearchRequest
		u []User
		e error
	}{
		{"1", SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[1], ss.users[2], ss.users[3]}, nil},
		{"2", SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[1]}, nil},
		{"3", SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByDesc}, []User{ss.users[1], ss.users[0], ss.users[2], ss.users[3]}, nil},
		{"4", SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[0]}, nil},
		{"5", SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[2], ss.users[3], ss.users[1]}, nil},
		{"6", SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}, nil},
		{"7", SearchRequest{Limit: 4, OrderField: "", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}, nil},
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
