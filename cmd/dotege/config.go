package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	envDebugKey                   = "DOTEGE_DEBUG"
	envDebugContainersValue       = "containers"
	envDebugHeadersValue          = "headers"
	envDebugHostnamesValue        = "hostnames"
	envSignalContainerKey         = "DOTEGE_SIGNAL_CONTAINER"
	envSignalContainerDefault     = ""
	envSignalTypeKey              = "DOTEGE_SIGNAL_TYPE"
	envSignalTypeDefault          = "HUP"
	envTemplateDestinationKey     = "DOTEGE_TEMPLATE_DESTINATION"
	envTemplateDestinationDefault = "/data/output/haproxy.cfg"
	envTemplateSourceKey          = "DOTEGE_TEMPLATE_SOURCE"
	envTemplateSourceDefault      = "./templates/haproxy.cfg.tpl"
	envUsersKey                   = "DOTEGE_USERS"
	envUsersDefault               = ""
	envProxyTagKey                = "DOTEGE_PROXYTAG"
	envProxyTagDefault            = ""
)

// Config is the user-definable configuration for Dotege.
type Config struct {
	Templates []TemplateConfig
	Signals   []ContainerSignal
	Users     []User
	ProxyTag  string

	DebugContainers bool
	DebugHeaders    bool
	DebugHostnames  bool
}

// User holds the details of a single user used for ACL purposes.
type User struct {
	Name     string   `yaml:"name"`
	Password string   `yaml:"password"`
	Groups   []string `yaml:"groups"`
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
	debug := toMap(splitList(strings.ToLower(optionalStringVar(envDebugKey, ""))))
	c := &Config{
		Templates: []TemplateConfig{
			{
				Source:      optionalStringVar(envTemplateSourceKey, envTemplateSourceDefault),
				Destination: optionalStringVar(envTemplateDestinationKey, envTemplateDestinationDefault),
			},
		},
		Signals:  createSignalConfig(),
		Users:    readUsers(),
		ProxyTag: optionalStringVar(envProxyTagKey, envProxyTagDefault),

		DebugContainers: debug[envDebugContainersValue],
		DebugHeaders:    debug[envDebugHeadersValue],
		DebugHostnames:  debug[envDebugHostnamesValue],
	}

	return c
}

func readUsers() []User {
	var users []User
	err := yaml.Unmarshal([]byte(optionalStringVar(envUsersKey, envUsersDefault)), &users)
	if err != nil {
		panic(fmt.Errorf("unable to parse users struct: %s", err))
	}
	return users
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

func toMap(input []string) map[string]bool {
	res := make(map[string]bool)
	for k := range input {
		res[input[k]] = true
	}
	return res
}
