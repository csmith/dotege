package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"time"
)

type DockerClient interface {
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

type ContainerMonitor struct {
	client DockerClient
}

type Operation int

const (
	Added = iota
	Removed
)

type ContainerEvent struct {
	Operation Operation
	Container Container
}

func (m ContainerMonitor) monitor(ctx context.Context, output chan<- ContainerEvent) error {
	ctx, cancel := context.WithCancel(ctx)
	stream, errors := m.startEventStream(ctx)
	timer := time.NewTimer(30 * time.Second)

	if err := m.publishExistingContainers(ctx, output); err != nil {
		cancel()
		return err
	}

	for {
		select {
		case event := <-stream:
			if event.Action == "create" {
				err, container := m.inspectContainer(ctx, event.Actor.ID)
				if err != nil {
					cancel()
					return err
				}
				output <- ContainerEvent{
					Operation: Added,
					Container: container,
				}
			} else {
				output <- ContainerEvent{
					Operation: Removed,
					Container: Container{
						Id: event.Actor.ID,
					},
				}
			}

		case err := <-errors:
			cancel()
			return err

		case <- timer.C:
			if err := m.publishExistingContainers(ctx, output); err != nil {
				cancel()
				return err
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (m ContainerMonitor) startEventStream(ctx context.Context) (<-chan events.Message, <-chan error) {
	args := filters.NewArgs()
	args.Add("type", "container")
	args.Add("event", "create")
	args.Add("event", "destroy")
	return m.client.Events(ctx, types.EventsOptions{Filters: args})
}

func (m ContainerMonitor) publishExistingContainers(ctx context.Context, output chan<- ContainerEvent) error {
	containers, err := m.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %s", err.Error())
	}

	for _, container := range containers {
		output <- ContainerEvent{
			Operation: Added,
			Container: Container{
				Id:     container.ID,
				Name:   container.Names[0][1:],
				Labels: container.Labels,
				Ports:  portsFromContainerPorts(container.Ports),
			},
		}
	}
	return nil
}

func (m ContainerMonitor) inspectContainer(ctx context.Context, id string) (error, Container) {
	container, err := m.client.ContainerInspect(ctx, id)
	if err != nil {
		return err, Container{}
	}

	return nil, Container{
		Id:     container.ID,
		Name:   container.Name[1:],
		Labels: container.Config.Labels,
		Ports:  portsFromContainerPortMap(container.HostConfig.PortBindings),
	}
}

// portsFromContainerPortMap collates all non-exposed TCP ports from the given map
func portsFromContainerPortMap(ps nat.PortMap) (ports []int) {
	for p, bindings := range ps {
		if p.Proto() == "tcp" && len(bindings) == 0 {
			ports = append(ports, p.Int())
		}
	}
	return
}

// portsFromContainerPorts collates all the non-exposed TCP ports from the given port list
func portsFromContainerPorts(ps []types.Port) (ports []int) {
	for _, p := range ps {
		if p.Type == "tcp" && p.PublicPort == 0 {
			ports = append(ports, int(p.PrivatePort))
		}
	}
	return
}
