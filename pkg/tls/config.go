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

func NewConfig() (*Config, error) {
	return &Config{
		OnUpdate: func() {},
	}, nil
}
