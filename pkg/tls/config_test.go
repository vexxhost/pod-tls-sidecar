package tls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	config, err := NewConfig()
	require.NoError(t, err)

	config.OnUpdate()
}
