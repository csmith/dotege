package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

var (
	config     *Config
	containers = make(Containers)
	GitSHA     string
)

func monitorSignals() <-chan bool {
	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		fmt.Printf("Received %s signal\n", sig)
		done <- true
	}()

	return done
}

func createTemplates(configs []TemplateConfig) Templates {
	var templates Templates
	for _, t := range configs {
		templates = append(templates, CreateTemplate(t.Source, t.Destination))
	}
	return templates
}

func main() {
	log.Printf("Dotege %s is starting", GitSHA)

	doneChan := monitorSignals()
	config = createConfig()

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	templates := createTemplates(config.Templates)

	containerMonitor := ContainerMonitor{client: dockerClient}

	jitterTimer := time.NewTimer(time.Minute)
	containerEvents := make(chan ContainerEvent)

	go func() {
		if err := containerMonitor.monitor(ctx, containerEvents); err != nil {
			log.Fatalf("Error monitoring containers: %v", err)
		}
	}()

	go func() {
		for {
			select {
			case event := <-containerEvents:
				switch event.Operation {
				case Added:
					if event.Container.Labels[labelProxyTag] == config.ProxyTag {
						log.Printf("Container added: %s (id: %s)", event.Container.Name, event.Container.Id)
						containers[event.Container.Id] = &event.Container
						jitterTimer.Reset(100 * time.Millisecond)
					} else {
						log.Printf(
							"Ignored container %s due to proxy tag (wanted: '%s', got: '%s')",
							event.Container.Name, config.ProxyTag, event.Container.Labels[labelProxyTag],
						)
					}
				case Removed:
					_, inExisting := containers[event.Container.Id]
					log.Printf(
						"Removed container with ID %s (was previously known: %t)",
						event.Container.Id,
						inExisting,
					)

					delete(containers, event.Container.Id)
					jitterTimer.Reset(100 * time.Millisecond)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-jitterTimer.C:
				updated := templates.Generate(struct {
					Containers map[string]*Container
					Hostnames  map[string]*Hostname
				}{
					containers,
					containers.Hostnames(),
				})

				if updated {
					signalContainer(dockerClient)
				}
			}
		}
	}()

	<-doneChan

	cancel()
	err = dockerClient.Close()
	if err != nil {
		panic(err)
	}
}

func signalContainer(dockerClient *client.Client) {
	for _, s := range config.Signals {
		var container *Container
		for _, c := range containers {
			if c.Name == s.Name {
				container = c
			}
		}

		if container != nil {
			log.Printf("Killing container %s (%s) with signal %s", container.Name, container.Id, s.Signal)
			err := dockerClient.ContainerKill(context.Background(), container.Id, s.Signal)
			if err != nil {
				log.Printf("Unable to send signal %s to container %s: %v", s.Signal, s.Name, err)
			}
		} else if config.ProxyTag != "" {
			log.Printf("Couldn't signal container %s as it is not known. Does it have the correct proxytag set?", s.Name)
		} else {
			log.Printf("Couldn't signal container %s as it is not known", s.Name)
		}
	}
}
