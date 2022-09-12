package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"localhost/coursera/hw3_bench/fast"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func FastSearch(out io.Writer) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	fast.FastSearch(out, data)
}

func FastSearchDefault(out io.Writer) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	fast.FastSearchDefault(out, data)
}

type benchResultsValues struct {
	name   string
	ops    uint32
	bytes  uint32
	allocs uint32
}

func main() {
	out, err := exec.Command("go", "test", "-bench", ".", "-benchmem").Output()
	if err != nil {
		panic(err)
	}

	benchs := []benchResultsValues{}

	reader := bufio.NewReader(bytes.NewReader(out))
	for {
		s, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if strings.Contains(s, "Benchmark") {
			ss := strings.Fields(s)
			ops, _ := strconv.Atoi(ss[2])
			bytes, _ := strconv.Atoi(ss[4])
			allocs, _ := strconv.Atoi(ss[6])
			bench := benchResultsValues{
				name:   ss[0],
				ops:    uint32(ops),
				bytes:  uint32(bytes),
				allocs: uint32(allocs),
			}
			benchs = append(benchs, bench)
		}
	}

	fmt.Printf("%s vs %s\n", benchs[0].name, benchs[1].name)
	fmt.Printf("\tns: \t%f\n", float32(benchs[0].ops)/float32(benchs[1].ops))
	fmt.Printf("\tmem: \t%f\n", float32(benchs[0].bytes)/float32(benchs[1].bytes))
	fmt.Printf("\talloc: \t%f\n", float32(benchs[0].allocs)/float32(benchs[1].allocs))
}
