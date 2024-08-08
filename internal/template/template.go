// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"os"
	"text/template"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/ghodss/yaml"
)

type Template struct {
	template *template.Template
}

func New(tmpl string) (*Template, error) {
	t, err := template.New("node-tls-sidecar").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	return &Template{template: t}, nil
}

func NewFromFile(path string) (*Template, error) {
	tmpl, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return New(string(tmpl))
}

func (t *Template) Execute(config *Values) (*cmv1.Certificate, error) {
	var buf bytes.Buffer
	err := t.template.Execute(&buf, config)
	if err != nil {
		return nil, err
	}

	spec := &cmv1.Certificate{}
	err = yaml.Unmarshal(buf.Bytes(), spec)
	if err != nil {
		return nil, err
	}

	return spec, nil
}
