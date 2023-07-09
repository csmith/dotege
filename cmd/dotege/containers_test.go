package main

import (
	"reflect"
	"testing"
)

func TestContainer_ShouldProxy(t *testing.T) {
	type fields struct {
		Id     string
		Name   string
		Labels map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"No labels", fields{Labels: map[string]string{}}, false},
		{"Only port", fields{Labels: map[string]string{labelProxy: "8080"}}, false},
		{"Vhost and port", fields{Labels: map[string]string{labelVhost: "example.com", labelProxy: "8080"}}, true},
		{"Vhost and invalid port", fields{Labels: map[string]string{labelVhost: "example.com", labelProxy: "bob"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Container{
				Id:     tt.fields.Id,
				Name:   tt.fields.Name,
				Labels: tt.fields.Labels,
			}
			if got := c.ShouldProxy(); got != tt.want {
				t.Errorf("ShouldProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainer_Port(t *testing.T) {
	tests := []struct {
		name      string
		container Container
		want      int
	}{
		{"No labels", Container{Labels: map[string]string{}}, -1},
		{"Text label", Container{Labels: map[string]string{labelProxy: "bob"}}, -1},
		{"Non-integer label", Container{Labels: map[string]string{labelProxy: "3.14159"}}, -1},
		{"Valid label", Container{Labels: map[string]string{labelProxy: "8080"}}, 8080},
		{"Negative", Container{Labels: map[string]string{labelProxy: "-100"}}, -1},
		{"Zero", Container{Labels: map[string]string{labelProxy: "0"}}, -1},
		{"Minimum", Container{Labels: map[string]string{labelProxy: "1"}}, 1},
		{"Maximum", Container{Labels: map[string]string{labelProxy: "65535"}}, 65535},
		{"Too high", Container{Labels: map[string]string{labelProxy: "65536"}}, -1},
		{"Bigger than int64", Container{Labels: map[string]string{labelProxy: "100000000000000000000"}}, -1},
		{"No labels, one port", Container{Labels: map[string]string{}, Ports: []int{123}}, 123},
		{"No labels, two ports", Container{Labels: map[string]string{}, Ports: []int{123, 456}}, -1},
		{"Label with one port", Container{Labels: map[string]string{labelProxy: "8080"}, Ports: []int{8081}}, 8080},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &tt.container
			if got := c.Port(); got != tt.want {
				t.Errorf("Port() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainer_Headers(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   map[string]string
	}{
		{"No labels", map[string]string{}, map[string]string{}},
		{"Other labels", map[string]string{"foo": "bar", "bar": "baz"}, map[string]string{}},
		{"Plain header", map[string]string{labelHeaders: "Foo: bar"}, map[string]string{"Foo": "bar"}},
		{"Plain header without colon", map[string]string{labelHeaders: "Foo bar"}, map[string]string{"Foo": "bar"}},
		{"Multiple headers", map[string]string{labelHeaders + ".1": "Foo bar", labelHeaders + ".2": "Baz: Quux"}, map[string]string{"Foo": "bar", "Baz": "Quux"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Container{
				Labels: tt.labels,
			}
			if got := c.Headers(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Headers() = %v, want %v", got, tt.want)
			}
		})
	}
}
