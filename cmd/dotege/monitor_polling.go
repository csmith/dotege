package main

import (
	"context"
	"github.com/moby/moby/client"
	"time"
)

type PollingDockerClient interface {
	ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
}

type PollingMonitor struct {
	client   PollingDockerClient
	interval time.Duration
}

func (p *PollingMonitor) Monitor(ctx context.Context, output chan<- ContainerEvent) error {
	seen := make(map[string]bool)

	for {
		seenThisTime := make(map[string]bool)

		res, err := p.client.ContainerList(ctx, client.ContainerListOptions{})
		if err != nil {
			return err
		}

		for i := range res.Items {
			seenThisTime[res.Items[i].ID] = true
			if _, ok := seen[res.Items[i].ID]; !ok {
				// We haven't seen the container before
				output <- ContainerEvent{
					Operation: Added,
					Container: Container{
						Id:     res.Items[i].ID,
						Name:   res.Items[i].Names[0][1:],
						Labels: res.Items[i].Labels,
						Ports:  portsFromContainerPorts(res.Items[i].Ports),
					},
				}

				seen[res.Items[i].ID] = true
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
