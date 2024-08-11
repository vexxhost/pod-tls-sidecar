// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/vexxhost/pod-tls-sidecar/internal/net"
	"github.com/vexxhost/pod-tls-sidecar/internal/podinfo"
)

type Values struct {
	PodInfo  podinfo.PodInfo
	Hostname string
	FQDN     string
}

func LoadValues() (*Values, error) {
	podInfo, err := podinfo.Load()
	if err != nil {
		return nil, err
	}

	hostname, err := net.Hostname()
	if err != nil {
		return nil, err
	}

	fqdn, err := net.FQDN()
	if err != nil {
		return nil, err
	}

	return &Values{
		PodInfo:  *podInfo,
		Hostname: hostname,
		FQDN:     fqdn,
	}, nil
}
