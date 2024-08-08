// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vexxhost/pod-tls-sidecar/internal/podinfo"
)

func TestTemplateFromString(t *testing.T) {
	tmpl, err := New(`
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: "{{ .PodInfo.Name }}-ssl"
  namespace: "{{ .PodInfo.Namespace }}"
spec:
  commonName: "{{ .FQDN }}"
  dnsNames:
    - "{{ .Hostname }}"
    - "{{ .FQDN }}"
  ipAddresses:
    - "{{ .PodInfo.IP }}"
  issuerRef:
    kind: ClusterIssuer
    name: atmosphere
  subject:
    countries:
      - CA
    localities:
      - Montreal
    organizationalUnits:
      - Cloud Infrastructure
    organizations:
      - VEXXHOST, Inc.
    provinces:
      - Quebec
  usages:
    - digital signature
    - key encipherment
  secretName: "{{ .PodInfo.Name }}-ssl"
`)
	require.NoError(t, err)

	values := &Values{
		PodInfo: podinfo.PodInfo{
			Name: "node-exporter-abcde",
			IP:   "172.16.1.10",
		},
		Hostname: "hostname",
		FQDN:     "fqdn",
	}

	certificate, err := tmpl.Execute(values)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s-ssl", values.PodInfo.Name), certificate.Name)
	assert.Equal(t, values.PodInfo.Namespace, certificate.Namespace)

	assert.Equal(t, values.FQDN, certificate.Spec.CommonName)

	assert.Len(t, certificate.Spec.DNSNames, 2)
	assert.Contains(t, certificate.Spec.DNSNames, values.Hostname)
	assert.Contains(t, certificate.Spec.DNSNames, values.FQDN)

	assert.Len(t, certificate.Spec.IPAddresses, 1)
	assert.Contains(t, certificate.Spec.IPAddresses, values.PodInfo.IP)

	assert.NotNil(t, certificate.Spec.Subject)
	assert.Len(t, certificate.Spec.Subject.Countries, 1)
	assert.Contains(t, certificate.Spec.Subject.Countries, "CA")
	assert.Len(t, certificate.Spec.Subject.Localities, 1)
	assert.Contains(t, certificate.Spec.Subject.Localities, "Montreal")
	assert.Len(t, certificate.Spec.Subject.OrganizationalUnits, 1)
	assert.Contains(t, certificate.Spec.Subject.OrganizationalUnits, "Cloud Infrastructure")
	assert.Len(t, certificate.Spec.Subject.Organizations, 1)
	assert.Contains(t, certificate.Spec.Subject.Organizations, "VEXXHOST, Inc.")
	assert.Len(t, certificate.Spec.Subject.Provinces, 1)
	assert.Contains(t, certificate.Spec.Subject.Provinces, "Quebec")

	assert.Len(t, certificate.Spec.Usages, 2)
	assert.Contains(t, certificate.Spec.Usages, cmv1.UsageDigitalSignature)
	assert.Contains(t, certificate.Spec.Usages, cmv1.UsageKeyEncipherment)

	assert.Equal(t, fmt.Sprintf("%s-ssl", values.PodInfo.Name), certificate.Spec.SecretName)
}

func TestTemplateFromFile(t *testing.T) {
	tmpl, err := NewFromFile("testdata/basic.yaml")
	require.NoError(t, err)

	values := &Values{
		PodInfo: podinfo.PodInfo{
			Name: "node-exporter-abcde",
			IP:   "172.16.1.10",
		},
		Hostname: "hostname",
		FQDN:     "fqdn",
	}

	certificate, err := tmpl.Execute(values)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s-ssl", values.PodInfo.Name), certificate.Name)
	assert.Equal(t, values.PodInfo.Namespace, certificate.Namespace)

	assert.Equal(t, values.FQDN, certificate.Spec.CommonName)

	assert.Len(t, certificate.Spec.DNSNames, 2)
	assert.Contains(t, certificate.Spec.DNSNames, values.Hostname)
	assert.Contains(t, certificate.Spec.DNSNames, values.FQDN)

	assert.Len(t, certificate.Spec.IPAddresses, 1)
	assert.Contains(t, certificate.Spec.IPAddresses, values.PodInfo.IP)

	assert.Len(t, certificate.Spec.Usages, 2)
	assert.Contains(t, certificate.Spec.Usages, cmv1.UsageClientAuth)
	assert.Contains(t, certificate.Spec.Usages, cmv1.UsageServerAuth)

	assert.Equal(t, fmt.Sprintf("%s-ssl", values.PodInfo.Name), certificate.Spec.SecretName)
}
