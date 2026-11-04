package main

import (
	"context"
	"fmt"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"time"
)

type StreamingDockerClient interface {
	Events(ctx context.Context, options client.EventsListOptions) client.EventsResult
	ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerInspect(
		ctx context.Context,
		containerID string,
		options client.ContainerInspectOptions,
	) (client.ContainerInspectResult, error)
}

// StreamingMonitor listens to events emitted via the docker client to determine when containers are created
// or stopped.
type StreamingMonitor struct {
	client         StreamingDockerClient
	resyncInterval time.Duration
}

func (m *StreamingMonitor) Monitor(ctx context.Context, output chan<- ContainerEvent) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := m.startEventStream(ctx)
	ticker := time.NewTicker(m.resyncInterval)
	defer ticker.Stop()

	if err := m.publishExistingContainers(ctx, output); err != nil {
		return err
	}

	for {
		select {
		case event := <-stream.Messages:
			if event.Action == "destroy" || event.Action == "rename" {
				output <- ContainerEvent{
					Operation: Removed,
					Container: Container{
						Id: event.Actor.ID,
					},
				}
			}

			if event.Action == "create" || event.Action == "rename" {
				err, c := m.inspectContainer(ctx, event.Actor.ID)
				if err != nil {
					return err
				}
				output <- ContainerEvent{
					Operation: Added,
					Container: c,
				}
			}

		case err := <-stream.Err:
			return err

		case <-ticker.C:
			if err := m.publishExistingContainers(ctx, output); err != nil {
				return err
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *StreamingMonitor) startEventStream(ctx context.Context) client.EventsResult {
	return m.client.Events(
		ctx,
		client.EventsListOptions{
			Filters: make(client.Filters).
				Add("type", "container").
				Add("event", "create", "destroy", "rename"),
		},
	)
}

func (m *StreamingMonitor) publishExistingContainers(ctx context.Context, output chan<- ContainerEvent) error {
	res, err := m.client.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %s", err.Error())
	}

	for _, c := range res.Items {
		output <- ContainerEvent{
			Operation: Added,
			Container: Container{
				Id:     c.ID,
				Name:   c.Names[0][1:],
				Labels: c.Labels,
				Ports:  portsFromContainerPorts(c.Ports),
			},
		}
	}
	return nil
}

func (m *StreamingMonitor) inspectContainer(ctx context.Context, id string) (error, Container) {
	res, err := m.client.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return err, Container{}
	}

	c := res.Container

	return nil, Container{
		Id:     c.ID,
		Name:   c.Name[1:],
		Labels: c.Config.Labels,
		Ports:  portsFromContainerPortMap(c.HostConfig.PortBindings),
	}
}

// portsFromContainerPortMap collates all non-exposed TCP ports from the given map
func portsFromContainerPortMap(ps network.PortMap) (ports []int) {
	for p, bindings := range ps {
		if p.Proto() == network.TCP && len(bindings) == 0 {
			ports = append(ports, int(p.Num()))
		}
	}
	return
}

// portsFromContainerPorts collates all the non-exposed TCP ports from the given port list
func portsFromContainerPorts(ps []container.PortSummary) (ports []int) {
	for _, p := range ps {
		if p.Type == "tcp" && p.PublicPort == 0 {
			ports = append(ports, int(p.PrivatePort))
		}
	}
	return
}
