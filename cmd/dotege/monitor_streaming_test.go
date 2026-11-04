package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
)

type fakeStreamingClient struct {
	lists atomic.Int32
}

func (f *fakeStreamingClient) Events(context.Context, client.EventsListOptions) client.EventsResult {
	return client.EventsResult{Messages: make(chan events.Message), Err: make(chan error)}
}

func (f *fakeStreamingClient) ContainerList(
	context.Context,
	client.ContainerListOptions,
) (client.ContainerListResult, error) {
	f.lists.Add(1)
	return client.ContainerListResult{}, nil
}

func (f *fakeStreamingClient) ContainerInspect(
	context.Context,
	string,
	client.ContainerInspectOptions,
) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{}, nil
}

func TestStreamingMonitorResyncsRepeatedly(t *testing.T) {
	fake := &fakeStreamingClient{}
	monitor := &StreamingMonitor{client: fake, resyncInterval: time.Millisecond}
	ctx := t.Context()

	go func() {
		_ = monitor.Monitor(ctx, make(chan ContainerEvent))
	}()

	assert.Eventually(
		t,
		func() bool { return fake.lists.Load() >= 3 },
		time.Second,
		time.Millisecond,
	)
}
