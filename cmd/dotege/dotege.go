package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/csmith/envflag"
	"github.com/docker/docker/client"
)

var (
	signalContainer     = flag.String("signal-container", "", "The container to signal when the template changes")
	signalType          = flag.String("signal-type", "HUP", "The signal to send to the container")
	templateSource      = flag.String("template-source", "./templates/haproxy.cfg.tpl", "The template to use")
	templateDestination = flag.String("template-destination", "/data/output/haproxy.cfg", "The destination to write the template to")
	proxyTag            = flag.String("proxytag", "", "If set, ignore any containers that do not have this tag as a label")

	containers = make(Containers)
	GitSHA     string
)

func main() {
	log.Printf("Dotege %s is starting", GitSHA)
	envflag.Parse(envflag.WithPrefix("DOTEGE_"))

	doneChan := monitorSignals()

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	templates := createTemplates()

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
					if event.Container.Labels[labelProxyTag] == *proxyTag {
						log.Printf("Container added: %s (id: %s)", event.Container.Name, event.Container.Id)
						containers[event.Container.Id] = &event.Container
						jitterTimer.Reset(100 * time.Millisecond)
					} else {
						log.Printf(
							"Ignored container %s due to proxy tag (wanted: '%s', got: '%s')",
							event.Container.Name, *proxyTag, event.Container.Labels[labelProxyTag],
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
					sendSignal(dockerClient)
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

func createTemplates() Templates {
	var templates Templates
	templates = append(templates, CreateTemplate(*templateSource, *templateDestination))
	return templates
}

func sendSignal(dockerClient *client.Client) {
	if *signalContainer == "" {
		return
	}

	var container *Container
	for _, c := range containers {
		if c.Name == *signalContainer {
			container = c
		}
	}

	if container != nil {
		log.Printf("Killing container %s (%s) with signal %s", container.Name, container.Id, *signalType)
		err := dockerClient.ContainerKill(context.Background(), container.Id, *signalType)
		if err != nil {
			log.Printf("Unable to send signal %s to container %s: %v", *signalType, *signalContainer, err)
		}
	} else if *proxyTag != "" {
		log.Printf("Couldn't signal container %s as it is not known. Does it have the correct proxytag set?", *signalContainer)
	} else {
		log.Printf("Couldn't signal container %s as it is not known", *signalContainer)
	}
}
