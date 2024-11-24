// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package podinfo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestLoad(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		os.Setenv("POD_UID", "test-uid")
		os.Setenv("POD_NAME", "test-name")
		os.Setenv("POD_NAMESPACE", "test-namespace")
		os.Setenv("POD_IP", "test-ip")

		info, err := Load()
		require.NoError(t, err)

		assert.Equal(t, types.UID("test-uid"), info.UID)
		assert.Equal(t, "test-name", info.Name)
		assert.Equal(t, "test-namespace", info.Namespace)
		assert.Equal(t, "test-ip", info.IP)
	})

	t.Run("error", func(t *testing.T) {
		os.Clearenv()

		_, err := Load()
		require.Error(t, err)
	})
}
