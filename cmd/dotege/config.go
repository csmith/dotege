package main

import (
	"os"
	"strings"
)

const (
	envSignalContainerKey         = "DOTEGE_SIGNAL_CONTAINER"
	envSignalContainerDefault     = ""
	envSignalTypeKey              = "DOTEGE_SIGNAL_TYPE"
	envSignalTypeDefault          = "HUP"
	envTemplateDestinationKey     = "DOTEGE_TEMPLATE_DESTINATION"
	envTemplateDestinationDefault = "/data/output/haproxy.cfg"
	envTemplateSourceKey          = "DOTEGE_TEMPLATE_SOURCE"
	envTemplateSourceDefault      = "./templates/haproxy.cfg.tpl"
	envProxyTagKey                = "DOTEGE_PROXYTAG"
	envProxyTagDefault            = ""
)

// Config is the user-definable configuration for Dotege.
type Config struct {
	Templates []TemplateConfig
	Signals   []ContainerSignal
	ProxyTag  string
}

// TemplateConfig configures a single template for the generator.
type TemplateConfig struct {
	Source      string
	Destination string
}

// ContainerSignal describes a container that should be sent a signal when the template output changes.
type ContainerSignal struct {
	Name   string
	Signal string
}

func optionalStringVar(key string, fallback string) (value string) {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = fallback
	}
	return
}

func createSignalConfig() []ContainerSignal {
	name := optionalStringVar(envSignalContainerKey, envSignalContainerDefault)
	if name == envSignalContainerDefault {
		return []ContainerSignal{}
	} else {
		return []ContainerSignal{
			{
				Name:   name,
				Signal: optionalStringVar(envSignalTypeKey, envSignalTypeDefault),
			},
		}
	}
}

func createConfig() *Config {
	c := &Config{
		Templates: []TemplateConfig{
			{
				Source:      optionalStringVar(envTemplateSourceKey, envTemplateSourceDefault),
				Destination: optionalStringVar(envTemplateDestinationKey, envTemplateDestinationDefault),
			},
		},
		Signals:  createSignalConfig(),
		ProxyTag: optionalStringVar(envProxyTagKey, envProxyTagDefault),
	}

	return c
}

func splitList(input string) (result []string) {
	result = []string{}
	for _, part := range strings.Split(strings.ReplaceAll(input, " ", ","), ",") {
		if len(part) > 0 {
			result = append(result, part)
		}
	}
	return
}
