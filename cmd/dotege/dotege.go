package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	loggers = struct {
		main       *zap.SugaredLogger
		headers    *zap.SugaredLogger
		hostnames  *zap.SugaredLogger
		containers *zap.SugaredLogger
	}{
		main:       createLogger(),
		headers:    zap.NewNop().Sugar(),
		hostnames:  zap.NewNop().Sugar(),
		containers: zap.NewNop().Sugar(),
	}

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

func createLogger() *zap.SugaredLogger {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.DisableCaller = true
	zapConfig.DisableStacktrace = true
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stdout"}
	logger, _ := zapConfig.Build()
	return logger.Sugar()
}

func createTemplates(configs []TemplateConfig) Templates {
	var templates Templates
	for _, t := range configs {
		templates = append(templates, CreateTemplate(t.Source, t.Destination))
	}
	return templates
}

func main() {
	loggers.main.Infof("Dotege %s is starting", GitSHA)

	doneChan := monitorSignals()
	config = createConfig()

	setUpDebugLoggers()

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
			loggers.main.Fatal("Error monitoring containers: ", err.Error())
		}
	}()

	go func() {
		for {
			select {
			case event := <-containerEvents:
				switch event.Operation {
				case Added:
					if event.Container.Labels[labelProxyTag] == config.ProxyTag {
						loggers.main.Debugf("Container added: %s", event.Container.Name)
						loggers.containers.Debugf("New container with name %s has id: %s", event.Container.Name, event.Container.Id)
						containers[event.Container.Id] = &event.Container
						jitterTimer.Reset(100 * time.Millisecond)
					} else {
						loggers.main.Debugf("Container ignored due to proxy tag: %s (wanted: '%s', got: '%s')", event.Container.Name, config.ProxyTag, event.Container.Labels[labelProxyTag])
					}
				case Removed:
					loggers.main.Debugf("Container removed: %s", event.Container.Id)

					_, inExisting := containers[event.Container.Id]
					loggers.containers.Debugf(
						"Removed container with ID %s, was in main containers: %t",
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
					Groups     []string
					Users      []User
				}{
					containers,
					containers.Hostnames(),
					groups(config.Users),
					config.Users,
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

func setUpDebugLoggers() {
	if config.DebugContainers {
		loggers.containers = loggers.main
	}

	if config.DebugHeaders {
		loggers.headers = loggers.main
	}

	if config.DebugHostnames {
		loggers.hostnames = loggers.main
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
			loggers.main.Debugf("Killing container %s (%s) with signal %s", container.Name, container.Id, s.Signal)
			err := dockerClient.ContainerKill(context.Background(), container.Id, s.Signal)
			if err != nil {
				loggers.main.Errorf("Unable to send signal %s to container %s: %s", s.Signal, s.Name, err.Error())
			}
		} else {
			loggers.main.Warnf("Couldn't signal container %s as it is not running", s.Name)
		}
	}
}

func groups(users []User) []string {
	groups := make(map[string]bool)
	for i := range users {
		for j := range users[i].Groups {
			groups[users[i].Groups[j]] = true
		}
	}

	var res []string
	for g := range groups {
		res = append(res, g)
	}
	return res
}
