package main

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	s, e := NewServer(AccessToken, "wrongdataset.xml")
	assert.Nil(t, s)
	assert.Error(t, e)

	s, e = NewServer(AccessToken, "hw4.md")
	assert.Nil(t, s)
	assert.Error(t, e)

}

func TestSearchRequest(t *testing.T) {
	s, e := NewServer(AccessToken, "dataset.xml")
	assert.NotNil(t, s)
	assert.NoError(t, e)

	tests := []struct {
		n string
		q string
		r *SearchRequest
		e error
	}{
		{"1", "&;#=?", nil, ErrorWrongQuery},
		{"2", "limit=-1", nil, ErrorWrongValue},
		{"3", "limit=30&offset=-1", nil, ErrorWrongValue},
		{"4", "limit=1", nil, ErrorMissedField},
		{"5", "offset=1", nil, ErrorMissedField},
		{"6", "offset=1&limit=1&order_by=ss", nil, ErrorStrIntCast},
		{"7", "offset=ss&limit=1&order_by=1", nil, ErrorStrIntCast},
		{"8", "offset=1&limit=ss&order_by=1", nil, ErrorStrIntCast},
		{"9", "offset=5&limit=1&order_by=1", nil, ErrorWrongValue},
	}
	for _, tt := range tests {
		t.Run(tt.n, func(t *testing.T) {
			_, err := s.SearchRequest(tt.q)
			assert.Equal(t, tt.e, errors.Cause(err))
		})
	}
}
