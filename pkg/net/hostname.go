// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package net

import (
	"os"
	"os/exec"
	"strings"
)

// Resolver provides hostname resolution.
type Resolver interface {
	Hostname() (string, error)
	FQDN() (string, error)
}

// SystemResolver implements Resolver using the real system hostname.
type SystemResolver struct {
	// HostnameCmd is the path to the hostname binary. Defaults to
	// "/bin/hostname" when empty.
	HostnameCmd string
}

func (r SystemResolver) hostnameCmd() string {
	if r.HostnameCmd != "" {
		return r.HostnameCmd
	}
	return "/bin/hostname"
}

func (SystemResolver) Hostname() (string, error) {
	return os.Hostname()
}

func (r SystemResolver) FQDN() (string, error) {
	cmd := exec.Command(r.hostnameCmd(), "--fqdn")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
