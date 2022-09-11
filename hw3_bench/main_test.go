package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

// запускаем перед основными функциями по разу чтобы файл остался в памяти в файловом кеше
// ioutil.Discard - это ioutil.Writer который никуда не пишет
func init() {
	SlowSearch(ioutil.Discard)
	FastSearch(ioutil.Discard)
}

// -----
// go test -v

func TestSearch(t *testing.T) {
	slowOut := new(bytes.Buffer)
	SlowSearch(slowOut)
	slowResult := slowOut.String()

	fastOut := new(bytes.Buffer)
	FastSearch(fastOut)
	fastResult := fastOut.String()

	if slowResult != fastResult {
		t.Errorf("results not match\nGot:\n%v\nExpected:\n%v", fastResult, slowResult)
	}
}

// -----
// go test -bench . -benchmem

func BenchmarkSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SlowSearch(ioutil.Discard)
	}
}

func BenchmarkFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FastSearch(ioutil.Discard)
	}
}

// goos: linux
// goarch: amd64
// pkg: localhost/coursera/hw3_bench
// cpu: AMD Ryzen 7 2700 Eight-Core Processor          
// BenchmarkSlow-8   	      27	  44724427 ns/op	19965092 B/op	  189812 allocs/op
// PASS
// ok  	localhost/coursera/hw3_bench	1.357s

// goos: linux
// goarch: amd64
// pkg: localhost/coursera/hw3_bench
// cpu: AMD Ryzen 7 2700 Eight-Core Processor          
// BenchmarkFast-8   	      32	  36022844 ns/op	16704014 B/op	  190781 allocs/op
// PASS
// ok  	localhost/coursera/hw3_bench	1.281s