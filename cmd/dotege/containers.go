package main

import (
	"strconv"
	"strings"
)

const (
	labelVhost    = "com.chameth.vhost"
	labelProxy    = "com.chameth.proxy"
	labelProxyTag = "com.chameth.proxytag"
	labelAuth     = "com.chameth.auth"
	labelHeaders  = "com.chameth.headers"
)

// Container describes a docker container that is running on the system.
type Container struct {
	Id     string
	Name   string
	Labels map[string]string
	Ports  []int
}

// ShouldProxy determines whether the container should be proxied to
func (c *Container) ShouldProxy() bool {
	_, hasVhost := c.Labels[labelVhost]
	hasPort := c.Port() > -1
	return hasPort && hasVhost
}

// Port returns the port the container accepts traffic on, or -1 if it could not be determined
func (c *Container) Port() int {
	l, ok := c.Labels[labelProxy]
	if ok {
		p, err := strconv.Atoi(l)

		if err != nil {
			loggers.main.Warnf("Invalid port specification on container %s: %s (%v)", c.Name, l, err)
			return -1
		}

		if p < 1 || p >= 1<<16 {
			loggers.main.Warnf("Invalid port specification on container %s: %s (out of range)", c.Name, l)
			return -1
		}

		return p
	}

	if len(c.Ports) == 1 {
		return c.Ports[0]
	}

	return -1
}

// Headers returns the list of headers that should be applied for this container
func (c *Container) Headers() map[string]string {
	res := make(map[string]string)
	for k, v := range c.Labels {
		if strings.HasPrefix(k, labelHeaders) {
			parts := strings.SplitN(v, " ", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(strings.TrimRight(parts[0], ":"))
				value := strings.TrimSpace(parts[1])
				res[name] = value
				loggers.headers.Debugf("Container %s has header %s => %s", c.Name, name, value)
			} else {
				loggers.main.Warnf("Container %s has invalid label %s (%s) - expecting name and value", c.Name, k, v)
			}
		}
	}
	return res
}

// Containers maps container IDs to their corresponding information
type Containers map[string]*Container

// Hostnames builds a mapping of primary hostnames to details about the containers that use them
func (c Containers) Hostnames() (hostnames map[string]*Hostname) {
	loggers.hostnames.Debugf("Calculating hostnames for %d containers", len(c))
	hostnames = make(map[string]*Hostname)
	for _, container := range c {
		if label, ok := container.Labels[labelVhost]; ok {
			names := splitList(label)
			primary := names[0]

			loggers.hostnames.Debugf(
				"Container %s (ID: %s) has vhosts: %s, port: %d, proxy status: %t",
				container.Name,
				container.Id,
				label,
				container.Port(),
				container.ShouldProxy(),
			)

			h := hostnames[primary]
			if h == nil {
				h = NewHostname(primary)
				hostnames[primary] = h
			}

			h.update(names[1:], container)
			loggers.hostnames.Debugf("Hostname %s now has %d containers and %d alternate names", h.Name, len(h.Containers), len(h.Alternatives))
		} else {
			loggers.hostnames.Debugf("Container %s (ID: %s) has no vhost label", container.Name, container.Id)
		}
	}
	return
}

// Hostname describes a hostname used for proxying.
type Hostname struct {
	Name         string
	Alternatives map[string]string
	Containers   []*Container
	Headers      map[string]string
	RequiresAuth bool
	AuthGroup    string
}

// NewHostname creates a new hostname with the given name
func NewHostname(name string) *Hostname {
	return &Hostname{
		Name:         name,
		Alternatives: make(map[string]string),
		Headers:      make(map[string]string),
	}
}

// update adds the alternate names and container information to the hostname
func (h *Hostname) update(alternates []string, container *Container) {
	h.Containers = append(h.Containers, container)

	for _, a := range alternates {
		h.Alternatives[a] = a
	}

	if label, ok := container.Labels[labelAuth]; ok {
		h.RequiresAuth = true
		h.AuthGroup = label
	}

	for k, v := range container.Headers() {
		loggers.headers.Debugf("Adding header for hostname %s: %s => %s", h.Name, k, v)
		h.Headers[k] = v
	}
}
