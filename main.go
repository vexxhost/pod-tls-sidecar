// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"

	"github.com/vexxhost/pod-tls-sidecar/internal/tls"
	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
)

func main() {
	var templateFile string
	pflag.StringVar(&templateFile, "template", "", "template file")

	var writePathConfig tls.WritePathConfig
	pflag.StringSliceVar(&writePathConfig.CertificateAuthorityPaths, "ca-path", []string{}, "certificate authority paths")
	pflag.StringSliceVar(&writePathConfig.CertificatePaths, "cert-path", []string{}, "certificate paths")
	pflag.StringSliceVar(&writePathConfig.CertificateKeyPaths, "key-path", []string{}, "certificate key paths")

	pflag.Parse()

	tmpl, err := template.NewFromFile(templateFile)
	if err != nil {
		log.Fatal(err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	manager, err := tls.NewManager(config, tmpl, &writePathConfig)
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
