package  main

import (
	"net/http"
	"context"
	"encoding/json"

)

func (srv *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context
	var in ProfileParams
	u, e := srv.Profile(ctx, in)
	js, _ := json.Marshal(u)
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

func (srv *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context
	var in CreateParams
	n, e := srv.Create(ctx, in)
	js, _ := json.Marshal(n)
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

func (srv *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context
	var in OtherCreateParams
	o, e := srv.Create(ctx, in)
	js, _ := json.Marshal(o)
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}


func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile": 
		srv.handleProfile(w, r)
	case "/user/create": 
		srv.handleCreate(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create": 
		srv.handleCreate(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
