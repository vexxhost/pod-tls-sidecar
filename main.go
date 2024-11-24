// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"

	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
	"github.com/vexxhost/pod-tls-sidecar/pkg/tls"
)

func main() {
	config, err := tls.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	var templateFile string
	pflag.StringVar(&templateFile, "template", "", "template file")

	pflag.StringSliceVar(&config.Paths.CertificateAuthorityPaths, "ca-path", []string{}, "certificate authority paths")
	pflag.StringSliceVar(&config.Paths.CertificatePaths, "cert-path", []string{}, "certificate paths")
	pflag.StringSliceVar(&config.Paths.CertificateKeyPaths, "key-path", []string{}, "certificate key paths")

	pflag.Parse()

	config.Template, err = template.NewFromFile(templateFile)
	if err != nil {
		log.Fatal(err)
	}

	config.RestConfig, err = rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	manager, err := tls.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	err = manager.Create(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("certificate created")

	go manager.Watch(ctx)

	select {}
}
