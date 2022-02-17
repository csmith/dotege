package main

import "testing"

func Test_domainsMatch(t *testing.T) {
	type args struct {
		domains1 []string
		domains2 []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"separate single domains", args{[]string{"example.com"}, []string{"example.org"}}, false},
		{"matching single domains", args{[]string{"example.com"}, []string{"example.com"}}, true},
		{"matching subject missing sans", args{[]string{"example.com", "example.org"}, []string{"example.com"}}, false},
		{"matching subject extra sans", args{[]string{"example.com"}, []string{"example.com", "example.org"}}, false},
		{"matching subject different sans", args{[]string{"example.com", "example.org"}, []string{"example.com", "example.net"}}, false},
		{"mismatched subject and san", args{[]string{"example.org", "example.com"}, []string{"example.com", "example.org"}}, true},
		{"reordered sans", args{[]string{"example.org", "example.com", "example.net"}, []string{"example.org", "example.net", "example.com"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domainsMatch(tt.args.domains1, tt.args.domains2); got != tt.want {
				t.Errorf("domainsMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
