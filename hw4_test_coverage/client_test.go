package main

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

const AccessToken string = "accesstoken"

type stubServer struct {
	time time.Duration
	data interface{}
}

func (s *stubServer) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestSearchClient_FindUsers(t *testing.T) {
	type fields struct {
		AccessToken string
		URL         string
	}
	type args struct {
		req SearchRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SearchResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &SearchClient{
				AccessToken: tt.fields.AccessToken,
				URL:         tt.fields.URL,
			}
			got, err := srv.FindUsers(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchClient.FindUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SearchClient.FindUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}
