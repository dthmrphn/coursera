package main

import (
	"io"
	"net/http"
	"os"
)

func hello(res http.ResponseWriter, req *http.Request) {
	res.Header().Set(
		"Content-Type",
		"text/html",
	)
	data, _ := os.ReadFile("dataset.xml")
	io.WriteString(res, string(data))
}
func main() {
	http.HandleFunc("/hello", hello)
	http.ListenAndServe(":9000", nil)
}
