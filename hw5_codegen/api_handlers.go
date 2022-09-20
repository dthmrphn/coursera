
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	
)
	
func (s *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		out interface{}
	)

	switch r.URL.Path {
	case "/user/profile":
		out, err = s.handleProfile(w, r)
	case "/user/create":
		out, err = s.handleCreate(w, r)
	default:
		err = ApiError{Err: fmt.Errorf("unknown method"), HTTPStatus: http.StatusNotFound}
	}

	response := struct {
		Data  interface{} `json:"response,omitempty"`
		Error string      `json:"error"`
	}{}

	if err == nil {
		response.Data = out
	} else {
		response.Error = err.Error()
		if errApi, ok := err.(ApiError); ok {
			w.WriteHeader(errApi.HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	
	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
	
func (s *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) (d interface{}, e error){
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := ioutil.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	in, err := NewProfileParams(params)
	if err != nil {
		return nil, err
	}

	return s.Profile(r.Context(), in)
}
	
func (s *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) (d interface{}, e error){
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	
	if r.Method != "POST" {
		return nil, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")}
	}
	
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := ioutil.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	in, err := NewCreateParams(params)
	if err != nil {
		return nil, err
	}

	return s.Create(r.Context(), in)
}
	
func (s *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		out interface{}
	)

	switch r.URL.Path {
	case "/user/create":
		out, err = s.handleCreate(w, r)
	default:
		err = ApiError{Err: fmt.Errorf("unknown method"), HTTPStatus: http.StatusNotFound}
	}

	response := struct {
		Data  interface{} `json:"response,omitempty"`
		Error string      `json:"error"`
	}{}

	if err == nil {
		response.Data = out
	} else {
		response.Error = err.Error()
		if errApi, ok := err.(ApiError); ok {
			w.WriteHeader(errApi.HTTPStatus)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	
	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
	
func (s *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) (d interface{}, e error){
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	
	if r.Method != "POST" {
		return nil, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")}
	}
	
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := ioutil.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	in, err := NewOtherCreateParams(params)
	if err != nil {
		return nil, err
	}

	return s.Create(r.Context(), in)
}
	
func NewProfileParams(p url.Values) (ProfileParams, error) {
	s := ProfileParams{}
	var err error = nil

	 //Login
	s.Login = p.Get("login")

	if s.Login == "" {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")}
	}

	return s, err
}
	
func NewCreateParams(p url.Values) (CreateParams, error) {
	s := CreateParams{}
	var err error = nil

	 //Login
	s.Login = p.Get("login")

	if s.Login == "" {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")}
	}

	if len(s.Login) < 10 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("login len must be >= 10")}
	}

	 //Name
	s.Name = p.Get("full_name")

	 //Status
	s.Status = p.Get("status")

	if s.Status == "" {
		s.Status = "user"
	}

	enumStatusValid := false
	enumStatus := []string{"user", "moderator", "admin"}
	for _, valid := range enumStatus {
		if valid == s.Status {
			enumStatusValid = true
			break
		}
	}
	if !enumStatusValid {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("status must be one of [%s]", strings.Join(enumStatus, ", "))}
	}

	 //Age
	s.Age, err = strconv.Atoi(p.Get("age"))
	if err != nil {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("age must be int")}
	}
	if s.Age < 0 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("age must be >= 0")}
	}

	if s.Age > 128 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("age must be <= 128")}
	}

	return s, err
}
	
func NewOtherCreateParams(p url.Values) (OtherCreateParams, error) {
	s := OtherCreateParams{}
	var err error = nil

	 //Username
	s.Username = p.Get("username")

	if s.Username == "" {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("username must me not empty")}
	}

	if len(s.Username) < 3 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("username len must be >= 3")}
	}

	 //Name
	s.Name = p.Get("account_name")

	 //Class
	s.Class = p.Get("class")

	if s.Class == "" {
		s.Class = "warrior"
	}

	enumClassValid := false
	enumClass := []string{"warrior", "sorcerer", "rouge"}
	for _, valid := range enumClass {
		if valid == s.Class {
			enumClassValid = true
			break
		}
	}
	if !enumClassValid {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("class must be one of [%s]", strings.Join(enumClass, ", "))}
	}

	 //Level
	s.Level, err = strconv.Atoi(p.Get("level"))
	if err != nil {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("level must be int")}
	}
	if s.Level < 1 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("level must be >= 1")}
	}

	if s.Level > 50 {
		return s, ApiError{http.StatusBadRequest, fmt.Errorf("level must be <= 50")}
	}

	return s, err
}
	