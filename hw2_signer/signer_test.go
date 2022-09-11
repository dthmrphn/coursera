package main

import (
	"testing"
)

func Test_singleHash(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"0", args{"0"}, "4108050209~502633748"},
		// {"1", args{"1"}, "2212294583~709660146"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := singleHash(tt.args.data); got != tt.want {
				t.Errorf("singleHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_multiHash(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"0", args{"4108050209~502633748"}, "29568666068035183841425683795340791879727309630931025356555"},
		// {"1", args{"2212294583~709660146"}, "4958044192186797981418233587017209679042592862002427381542"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := multiHash(tt.args.data); got != tt.want {
				t.Errorf("multiHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
