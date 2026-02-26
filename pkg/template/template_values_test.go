// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgnet "github.com/vexxhost/pod-tls-sidecar/pkg/net"
)

func TestLoadValues(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Setenv("POD_UID", "test-uid")
		t.Setenv("POD_NAME", "test-name")
		t.Setenv("POD_NAMESPACE", "test-namespace")
		t.Setenv("POD_IP", "10.0.0.1")

		values, err := LoadValues()
		require.NoError(t, err)

		assert.Equal(t, "test-uid", string(values.PodInfo.UID))
		assert.Equal(t, "test-name", values.PodInfo.Name)
		assert.Equal(t, "test-namespace", values.PodInfo.Namespace)
		assert.Equal(t, "10.0.0.1", values.PodInfo.IP)
		assert.NotEmpty(t, values.Hostname)
		assert.NotEmpty(t, values.FQDN)
	})

	t.Run("podinfo error", func(t *testing.T) {
		// Unset the required POD_* env vars so podinfo.Load returns an error.
		keys := []string{"POD_UID", "POD_NAME", "POD_NAMESPACE", "POD_IP"}
		saved := make(map[string]string, len(keys))
		hadKey := make(map[string]bool, len(keys))
		for _, k := range keys {
			saved[k], hadKey[k] = os.LookupEnv(k)
			os.Unsetenv(k)
		}
		t.Cleanup(func() {
			for _, k := range keys {
				if hadKey[k] {
					os.Setenv(k, saved[k])
				} else {
					os.Unsetenv(k)
				}
			}
		})

		_, err := LoadValues()
		require.Error(t, err)
	})

	t.Run("hostname error", func(t *testing.T) {
		t.Setenv("POD_UID", "test-uid")
		t.Setenv("POD_NAME", "test-name")
		t.Setenv("POD_NAMESPACE", "test-namespace")
		t.Setenv("POD_IP", "10.0.0.1")

		orig := pkgnet.HostnameFunc()
		pkgnet.SetHostnameFunc(func() (string, error) {
			return "", fmt.Errorf("hostname error")
		})
		defer pkgnet.SetHostnameFunc(orig)

		_, err := LoadValues()
		require.Error(t, err)
	})

	t.Run("fqdn error", func(t *testing.T) {
		t.Setenv("POD_UID", "test-uid")
		t.Setenv("POD_NAME", "test-name")
		t.Setenv("POD_NAMESPACE", "test-namespace")
		t.Setenv("POD_IP", "10.0.0.1")

		orig := pkgnet.FqdnCmd()
		pkgnet.SetFqdnCmd("/nonexistent/hostname")
		defer pkgnet.SetFqdnCmd(orig)

		_, err := LoadValues()
		require.Error(t, err)
	})
}
