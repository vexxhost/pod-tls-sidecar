package tls

import (
	"time"

	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/vexxhost/pod-tls-sidecar/pkg/net"
	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
	kubernetes "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type Config struct {
	RestConfig *rest.Config
	Template   *template.Template
	Paths      *WritePathConfig
	OnUpdate   func()

	// Resolver provides hostname resolution. If nil, net.SystemResolver{} is used.
	Resolver net.Resolver

	// WatchRetryDelay is the time between watch reconnect attempts.
	// If zero, defaults to 5 seconds.
	WatchRetryDelay time.Duration

	// SecretClient is an optional Kubernetes secret client. If nil, one is
	// created from RestConfig. Useful for injecting fakes in tests.
	SecretClient kubernetes.SecretInterface

	// CertificateClient is an optional cert-manager certificate client. If
	// nil, one is created from RestConfig. Useful for injecting fakes in tests.
	CertificateClient cmclient.CertificateInterface
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
