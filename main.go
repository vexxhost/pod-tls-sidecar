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
	var templateFile string
	var caPaths, certPaths, keyPaths []string

	pflag.StringVar(&templateFile, "template", "", "template file")
	pflag.StringSliceVar(&caPaths, "ca-path", []string{}, "certificate authority paths")
	pflag.StringSliceVar(&certPaths, "cert-path", []string{}, "certificate paths")
	pflag.StringSliceVar(&keyPaths, "key-path", []string{}, "certificate key paths")

	pflag.Parse()

	tmpl, err := template.NewFromFile(templateFile)
	if err != nil {
		log.Fatal(err)
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	paths := &tls.WritePathConfig{
		CertificateAuthorityPaths: caPaths,
		CertificatePaths:          certPaths,
		CertificateKeyPaths:       keyPaths,
	}

	config, err := tls.NewConfig(
		tls.WithTemplate(tmpl),
		tls.WithRestConfig(restConfig),
		tls.WithPaths(paths),
	)
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

	go manager.Watch(ctx)

	select {}
}
