// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package podinfo

import (
	"github.com/kelseyhightower/envconfig"
	"k8s.io/apimachinery/pkg/types"
)

type PodInfo struct {
	UID       types.UID `envconfig:"UID" required:"true"`
	Name      string    `envconfig:"NAME" required:"true"`
	Namespace string    `envconfig:"NAMESPACE" required:"true"`
	IP        string    `envconfig:"IP" required:"true"`
}

func Load() (*PodInfo, error) {
	var info PodInfo
	err := envconfig.Process("POD", &info)

	return &info, err
}
