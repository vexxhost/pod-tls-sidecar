// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestWriteCallsOnUpdateOnlyWhenFilesChange(t *testing.T) {
	dir := t.TempDir()
	updates := 0

	manager := &Manager{
		config: &Config{
			Paths: &WritePathConfig{
				CertificateAuthorityPaths: []string{filepath.Join(dir, "ca.crt")},
				CertificatePaths:          []string{filepath.Join(dir, "tls.crt")},
				CertificateKeyPaths:       []string{filepath.Join(dir, "tls.key")},
			},
			OnUpdate: func() {
				updates++
			},
		},
		logger: log.NewEntry(log.New()),
	}

	secret := &v1.Secret{
		Data: map[string][]byte{
			"ca.crt":  []byte("ca"),
			"tls.crt": []byte("cert"),
			"tls.key": []byte("key"),
		},
	}

	manager.write(secret)
	assert.Equal(t, 1, updates)

	manager.write(secret)
	assert.Equal(t, 1, updates)

	secret.Data["tls.crt"] = []byte("rotated-cert")
	manager.write(secret)
	assert.Equal(t, 2, updates)
}

func TestWriteFileReportsWhetherFileChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tls.crt")

	manager := &Manager{
		logger: log.NewEntry(log.New()),
	}

	assert.True(t, manager.writeFile(path, []byte("cert")))
	assert.False(t, manager.writeFile(path, []byte("cert")))
	assert.True(t, manager.writeFile(path, []byte("rotated-cert")))

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("rotated-cert"), data)
}
