package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
	"k8s.io/client-go/rest"
)

func TestWithRestConfig(t *testing.T) {
	restConfig := &rest.Config{}
	config, err := NewConfig(WithRestConfig(restConfig))
	require.NoError(t, err)

	assert.Equal(t, restConfig, config.RestConfig)
}

func TestWithTemplate(t *testing.T) {
	tmpl := &template.Template{}
	config, err := NewConfig(WithTemplate(tmpl))
	require.NoError(t, err)

	assert.Equal(t, tmpl, config.Template)
}

func TestWithPaths(t *testing.T) {
	paths := &WritePathConfig{}
	config, err := NewConfig(WithPaths(paths))
	require.NoError(t, err)

	assert.Equal(t, paths, config.Paths)
}

func TestWithOnUpdate(t *testing.T) {
	var called bool
	onUpdate := func() { called = true }
	config, err := NewConfig(WithOnUpdate(onUpdate))
	require.NoError(t, err)

	config.OnUpdate()
	assert.True(t, called)
}

func TestNewConfigMultipleOptions(t *testing.T) {
	restConfig := &rest.Config{}
	tmpl := &template.Template{}
	paths := &WritePathConfig{}
	var called bool
	onUpdate := func() { called = true }

	config, err := NewConfig(
		WithRestConfig(restConfig),
		WithTemplate(tmpl),
		WithPaths(paths),
		WithOnUpdate(onUpdate),
	)
	require.NoError(t, err)

	assert.Equal(t, restConfig, config.RestConfig)
	assert.Equal(t, tmpl, config.Template)
	assert.Equal(t, paths, config.Paths)

	config.OnUpdate()
	assert.True(t, called)
}

func TestNewConfigWithoutOnUpdate(t *testing.T) {
	config, err := NewConfig()
	require.NoError(t, err)

	assert.NotNil(t, config.OnUpdate)
	assert.NotPanics(t, func() {
		config.OnUpdate()
	})
}
