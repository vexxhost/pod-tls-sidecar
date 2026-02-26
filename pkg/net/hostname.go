// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package net

import (
	"os"
	"os/exec"
	"strings"
)

// hostnameFunc is the function used to obtain the system hostname. It is a
// variable so that tests can override it to exercise the error path.
var hostnameFunc = os.Hostname

// fqdnCmd is the path to the hostname binary used by FQDN; it is a variable
// so that tests can override it to exercise the error path.
var fqdnCmd = "/bin/hostname"

// HostnameFunc returns the current hostnameFunc. Exposed for tests in other
// packages that need to override and restore it.
func HostnameFunc() func() (string, error) { return hostnameFunc }

// SetHostnameFunc replaces hostnameFunc. Exposed for tests in other packages.
func SetHostnameFunc(fn func() (string, error)) { hostnameFunc = fn }

// FqdnCmd returns the current fqdnCmd value. Exposed for tests in other
// packages that need to override and restore it.
func FqdnCmd() string { return fqdnCmd }

// SetFqdnCmd replaces the fqdnCmd value. Exposed for tests in other packages.
func SetFqdnCmd(cmd string) { fqdnCmd = cmd }

func Hostname() (string, error) {
	return hostnameFunc()
}

func FQDN() (string, error) {
	cmd := exec.Command(fqdnCmd, "--fqdn")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
