package main

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"golang.org/x/net/context"
	"time"
)

type PollingDockerClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
}

type PollingMonitor struct {
	client   PollingDockerClient
	interval time.Duration
}

func (p *PollingMonitor) Monitor(ctx context.Context, output chan<- ContainerEvent) error {
	seen := make(map[string]bool)

	for {
		seenThisTime := make(map[string]bool)

		res, err := p.client.ContainerList(ctx, container.ListOptions{})
		if err != nil {
			return err
		}

		for i := range res {
			seenThisTime[res[i].ID] = true
			if _, ok := seen[res[i].ID]; !ok {
				// We haven't seen the container before
				output <- ContainerEvent{
					Operation: Added,
					Container: Container{
						Id:     res[i].ID,
						Name:   res[i].Names[0][1:],
						Labels: res[i].Labels,
						Ports:  portsFromContainerPorts(res[i].Ports),
					},
				}

				seen[res[i].ID] = true
			}
		}

		for i := range seen {
			if _, ok := seenThisTime[i]; !ok {
				// We didn't see this container, it must have gone
				delete(seen, i)
				output <- ContainerEvent{
					Operation: Removed,
					Container: Container{
						Id: i,
					},
				}
			}
		}

		time.Sleep(p.interval)
	}
}
