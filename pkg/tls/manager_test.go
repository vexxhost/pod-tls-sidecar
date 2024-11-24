package tls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoCrashWithoutHook(t *testing.T) {
	config, err := NewConfig()
	require.NoError(t, err)

	config.OnUpdate()
}
