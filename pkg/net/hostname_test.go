// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package net

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemResolver_Hostname(t *testing.T) {
	r := SystemResolver{}
	hostname, err := r.Hostname()
	require.NoError(t, err)
	assert.NotEmpty(t, hostname)
}

func TestSystemResolver_FQDN(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := SystemResolver{}
		fqdn, err := r.FQDN()
		require.NoError(t, err)
		assert.NotEmpty(t, fqdn)
	})

	t.Run("error", func(t *testing.T) {
		r := SystemResolver{HostnameCmd: "/nonexistent/hostname"}
		_, err := r.FQDN()
		require.Error(t, err)
	})
}

func TestSystemResolver_hostnameCmd(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		r := SystemResolver{}
		assert.Equal(t, "/bin/hostname", r.hostnameCmd())
	})

	t.Run("custom", func(t *testing.T) {
		r := SystemResolver{HostnameCmd: "/usr/bin/hostname"}
		assert.Equal(t, "/usr/bin/hostname", r.hostnameCmd())
	})
}
