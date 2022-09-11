package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	mu sync.Mutex
)

func dataSignerMd5(data string) string {
	mu.Lock()
	defer mu.Unlock()
	return DataSignerMd5(data)
}

func dataSignerCrc32(data ...string) []string {
	rv := make([]string, len(data))
	wg := sync.WaitGroup{}
	wg.Add(len(data))
	for i := range data {
		go func(i int) {
			rv[i] = DataSignerCrc32(data[i])
			wg.Done()
		}(i)
	}
	wg.Wait()
	return rv
}

func singleHash(data string) string {
	return strings.Join(dataSignerCrc32(data, dataSignerMd5(data)), "~")
}

func SingleHash(in, out chan interface{}) {
	var tohash string
	wg := sync.WaitGroup{}
	for i := range in {
		switch m := (i).(type) {
		case int:
			tohash = fmt.Sprintf("%d", m)
		case string:
			tohash = m
		}
		wg.Add(1)
		go func(str string) {
			out <- singleHash(str)
			wg.Done()
		}(tohash)
	}
	wg.Wait()
}

func multiHash(data string) string {
	datas := make([]string, 6)

	for i := range datas {
		datas[i] = fmt.Sprintf("%d%s", i, data)
	}

	return strings.Join(dataSignerCrc32(datas...), "")
}

func MultiHash(in, out chan interface{}) {
	var tohash string
	wg := sync.WaitGroup{}
	for i := range in {
		switch m := (i).(type) {
		case int:
			tohash = fmt.Sprintf("%d", m)
		case string:
			tohash = m
		}
		wg.Add(1)
		go func(str string) {
			out <- multiHash(str)
			wg.Done()
		}(tohash)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	data := make([]string, 0)

	for i := range in {
		switch m := (i).(type) {
		case string:
			data = append(data, m)
		}
	}
	sort.Strings(data)
	out <- fmt.Sprint(strings.Join(data, "_"))
}

func ExecutePipeline(jobs ...job) {
	chs := make([]chan interface{}, len(jobs)+1)
	for i := 0; i < len(jobs)+1; i++ {
		chs[i] = make(chan interface{})
	}
	wg := sync.WaitGroup{}
	for i := range jobs {
		wg.Add(1)
		go func(i int, j job, in, out chan interface{}) {
			j(in, out)
			wg.Done()
			close(out)
		}(i, jobs[i], chs[i], chs[i+1])
	}
	wg.Wait()
}

func main() {
	in := make(chan interface{}, 1)
	out := make(chan interface{}, 1)

	in <- "1"

	SingleHash(in, out)
	MultiHash(out, in)
	CombineResults(in, out)
	fmt.Println((<-out).(string))
}
