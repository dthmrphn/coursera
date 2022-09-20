package main

import (
	"bytes"
	"io"
	"testing"
)

type testCase struct {
	name string
	mhs  []*MethodHandler
}

var tests []testCase

func init() {
	tests = []testCase{
		{"1", []*MethodHandler{
			&MethodHandler{
				n:    "",
				rec:  "*MyApi",
				arg:  "srv",
				args: []string(nil),
				argt: []string(nil),
				rets: []string(nil),
				rett: []string(nil),
				url:  "",
				child: []*MethodHandler{
					&MethodHandler{
						n:    "Profile",
						rec:  "*MyApi",
						arg:  "srv",
						args: []string{"ctx", "in"},
						argt: []string{"context.Context", "ProfileParams"},
						rets: []string{"*User", "error"},
						rett: []string{"u", "e"},
						url:  "/user/profile",
						// child:MethodHandler(nil),
					},
					&MethodHandler{
						n:    "Create",
						rec:  "*MyApi",
						arg:  "srv",
						args: []string{"ctx", "in"},
						argt: []string{"context.Context", "CreateParams"},
						rets: []string{"*NewUser", "error"},
						rett: []string{"n", "e"},
						url:  "/user/create",
						// child:[]MethodHandler(nil),
					},
				},
			},
		},
		},
	}
}

func TestWriteHttpHandlersTemplate(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			WriteHttpHandlersTemplate(w, tt.mhs)
		})
	}
}

func BenchmarkTemplate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WriteHttpHandlersTemplate(io.Discard, tests[0].mhs)
	}
}

func BenchmarkFormat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WriteHttpHandlersFormat(io.Discard, tests[0].mhs)
	}
}
