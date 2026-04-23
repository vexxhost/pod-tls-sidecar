// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/vexxhost/pod-tls-sidecar/pkg/net"
	"github.com/vexxhost/pod-tls-sidecar/pkg/podinfo"
)

type Values struct {
	PodInfo  podinfo.PodInfo
	Hostname string
	FQDN     string
}

func LoadValues(resolver net.Resolver) (*Values, error) {
	podInfo, err := podinfo.Load()
	if err != nil {
		return nil, err
	}

	hostname, err := resolver.Hostname()
	if err != nil {
		return nil, err
	}

	fqdn, err := resolver.FQDN()
	if err != nil {
		return nil, err
	}

	return &Values{
		PodInfo:  *podInfo,
		Hostname: hostname,
		FQDN:     fqdn,
	}, nil
}
