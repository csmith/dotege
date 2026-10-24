package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
)

type fakeStreamingClient struct {
	lists atomic.Int32
}

func (f *fakeStreamingClient) Events(context.Context, events.ListOptions) (<-chan events.Message, <-chan error) {
	return make(chan events.Message), make(chan error)
}

func (f *fakeStreamingClient) ContainerList(context.Context, container.ListOptions) ([]types.Container, error) {
	f.lists.Add(1)
	return nil, nil
}

func (f *fakeStreamingClient) ContainerInspect(context.Context, string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

func TestStreamingMonitorResyncsRepeatedly(t *testing.T) {
	client := &fakeStreamingClient{}
	monitor := &StreamingMonitor{client: client, resyncInterval: time.Millisecond}
	ctx := t.Context()

	go func() {
		_ = monitor.Monitor(ctx, make(chan ContainerEvent))
	}()

	assert.Eventually(
		t,
		func() bool { return client.lists.Load() >= 3 },
		time.Second,
		time.Millisecond,
	)
}
