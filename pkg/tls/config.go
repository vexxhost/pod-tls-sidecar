package tls

import (
	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
	"k8s.io/client-go/rest"
)

type Config struct {
	RestConfig *rest.Config
	Template   *template.Template
	Paths      *WritePathConfig
	OnUpdate   func()
}

type Option func(*Config)

func WithRestConfig(restConfig *rest.Config) Option {
	return func(c *Config) {
		c.RestConfig = restConfig
	}
}

func WithTemplate(template *template.Template) Option {
	return func(c *Config) {
		c.Template = template
	}
}

func WithPaths(paths *WritePathConfig) Option {
	return func(c *Config) {
		c.Paths = paths
	}
}

func WithOnUpdate(onUpdate func()) Option {
	return func(c *Config) {
		c.OnUpdate = onUpdate
	}
}

func NewConfig(opts ...Option) (*Config, error) {
	config := &Config{
		OnUpdate: func() {},
	}

	for _, opt := range opts {
		opt(config)
	}

	return config, nil
}
