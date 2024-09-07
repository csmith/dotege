package main

import "golang.org/x/net/context"

type Operation int

const (
	Added Operation = iota
	Removed
)

type ContainerEvent struct {
	Operation Operation
	Container Container
}

type Monitor interface {
	Monitor(ctx context.Context, output chan<- ContainerEvent) error
}
