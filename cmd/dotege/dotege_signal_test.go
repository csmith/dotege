package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendSignalTargetsConfiguredContainer(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "name", target: "configured-name"},
		{name: "id", target: "0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotSignal string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotSignal = r.URL.Query().Get("signal")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion("1.41"))
			require.NoError(t, err)
			defer dockerClient.Close()

			oldTarget, oldSignal, oldTag := *signalContainer, *signalType, *proxyTag
			defer func() { *signalContainer, *signalType, *proxyTag = oldTarget, oldSignal, oldTag }()
			*signalContainer, *signalType, *proxyTag = tt.target, "USR1", "required-tag"
			oldContainers := containers
			t.Cleanup(func() { containers = oldContainers })
			containers = Containers{}

			sendSignal(dockerClient)

			assert.Equal(t, "/v1.41/containers/"+tt.target+"/kill", gotPath)
			assert.Equal(t, "USR1", gotSignal)
		})
	}
}

func TestSendSignalEmptyTargetIsNoOp(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()
	dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion("1.41"))
	require.NoError(t, err)
	defer dockerClient.Close()

	oldTarget := *signalContainer
	defer func() { *signalContainer = oldTarget }()
	*signalContainer = ""
	sendSignal(dockerClient)
	assert.False(t, called)
}

func TestSendSignalLogsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	defer server.Close()
	dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion("1.41"))
	require.NoError(t, err)
	defer dockerClient.Close()

	oldTarget, oldSignal := *signalContainer, *signalType
	defer func() { *signalContainer, *signalType = oldTarget, oldSignal }()
	*signalContainer, *signalType = "missing", "HUP"

	assert.NotPanics(t, func() { sendSignal(dockerClient) })
}

func TestSendSignalUsesConfiguredTargetEvenWhenNotCached(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion("1.41"))
	require.NoError(t, err)
	defer dockerClient.Close()

	oldTarget, oldTag := *signalContainer, *proxyTag
	defer func() { *signalContainer, *proxyTag = oldTarget, oldTag }()
	*signalContainer, *proxyTag = "uncached", "different"
	oldContainers := containers
	t.Cleanup(func() { containers = oldContainers })
	containers = Containers{"cached": &Container{Name: "other", Id: "cached"}}

	sendSignal(dockerClient)
	assert.Equal(t, fmt.Sprintf("/v1.41/containers/%s/kill", *signalContainer), gotPath)
}
