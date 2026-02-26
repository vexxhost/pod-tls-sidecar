// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package net

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessors(t *testing.T) {
	// HostnameFunc / SetHostnameFunc
	origHF := hostnameFunc
	fn := func() (string, error) { return "test", nil }
	SetHostnameFunc(fn)
	assert.NotNil(t, HostnameFunc())
	hostnameFunc = origHF

	// FqdnCmd / SetFqdnCmd
	origCmd := fqdnCmd
	SetFqdnCmd("/tmp/custom-hostname")
	assert.Equal(t, "/tmp/custom-hostname", FqdnCmd())
	fqdnCmd = origCmd
}

func TestHostname(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		hostname, err := Hostname()
		require.NoError(t, err)
		assert.NotEmpty(t, hostname)
	})

	t.Run("error", func(t *testing.T) {
		orig := hostnameFunc
		hostnameFunc = func() (string, error) { return "", fmt.Errorf("hostname error") }
		defer func() { hostnameFunc = orig }()

		_, err := Hostname()
		require.Error(t, err)
	})
}

func TestFQDN(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fqdn, err := FQDN()
		require.NoError(t, err)
		assert.NotEmpty(t, fqdn)
	})

	t.Run("error", func(t *testing.T) {
		orig := fqdnCmd
		fqdnCmd = "/nonexistent/hostname"
		defer func() { fqdnCmd = orig }()

		_, err := FQDN()
		require.Error(t, err)
	})
}
