// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/vexxhost/pod-tls-sidecar/internal/template"
)

type PodConfig struct {
	UID       string `envconfig:"POD_UID" required:"true"`
	Name      string `envconfig:"POD_NAME" required:"true"`
	Namespace string `envconfig:"POD_NAMESPACE" required:"true"`
	IP        string `envconfig:"POD_IP" required:"true"`
}

type WritePathConfig struct {
	CertificateAuthorityPaths []string
	CertificatePaths          []string
	CertificateKeyPaths       []string
}

type Manager struct {
	certificate       *cmv1.Certificate
	certificateClient cmclient.CertificateInterface
	logger            *log.Entry
	paths             *WritePathConfig
	secretClient      kubernetes.SecretInterface
}

func NewManager(config *rest.Config, tmpl *template.Template, paths *WritePathConfig) (*Manager, error) {
	values, err := template.LoadValues()
	if err != nil {
		return nil, err
	}

	certificate, err := tmpl.Execute(values)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cmClient, err := cmclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	mgr := &Manager{
		certificate:       certificate,
		certificateClient: cmClient.Certificates(certificate.Namespace),
		logger: log.WithFields(log.Fields{
			"certificateName": certificate.Name,
			"podName":         values.PodInfo.Name,
			"podNamespace":    values.PodInfo.Namespace,
			"podUID":          values.PodInfo.UID,
			"podIP":           values.PodInfo.IP,
			"hostname":        values.Hostname,
			"fqdn":            values.FQDN,
		}),
		paths:        paths,
		secretClient: clientset.Secrets(certificate.Namespace),
	}

	return mgr, nil
}

func (m *Manager) Create(ctx context.Context) error {
	// Create certificate
	_, err := m.certificateClient.Create(ctx, m.certificate, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	m.logger.Info("certificate created")

	// Wait for certificate to become ready
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 300*time.Second, true, func(ctx context.Context) (bool, error) {
		certificate, err := m.certificateClient.Get(ctx, m.certificate.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range certificate.Status.Conditions {
			if condition.Type == cmv1.CertificateConditionReady {
				if condition.Status == cmmeta.ConditionTrue {
					return true, nil
				}

				m.logger.WithFields(log.Fields{
					"reason":  condition.Reason,
					"message": condition.Message,
				}).Info("certificate not ready")
			}
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	m.logger.Info("certificate ready")

	// Create patch with ownerReference so the secret is garbage collected
	patch := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/metadata/ownerReferences",
			"value": m.certificate.OwnerReferences,
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	m.logger.Info("patching secret")

	// Patch secret with ownerReference
	_, err = m.secretClient.Patch(ctx, m.certificate.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (m *Manager) Watch(ctx context.Context) {
	for {
		m.watch(ctx)
		m.logger.Info("watch closed or disconnected, retrying in 5 seconds")

		time.Sleep(5 * time.Second)
	}
}

func (m *Manager) watch(ctx context.Context) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", m.certificate.Name).String()

	listWatcher := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return m.secretClient.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return m.secretClient.Watch(ctx, options)
		},
	}

	_, controller := cache.NewInformer(
		listWatcher,
		&v1.Secret{},
		time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				secret := obj.(*v1.Secret)
				m.write(secret)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				secret := newObj.(*v1.Secret)
				m.write(secret)
			},
			DeleteFunc: func(obj interface{}) {
				m.logger.Fatal("secret deleted")
			},
		},
	)

	stop := make(chan struct{})
	defer close(stop)
	controller.Run(stop)
}

func (m *Manager) write(secret *v1.Secret) {
	for _, path := range m.paths.CertificateAuthorityPaths {
		m.writeFile(path, secret.Data["ca.crt"])
	}

	for _, path := range m.paths.CertificatePaths {
		m.writeFile(path, secret.Data["tls.crt"])
	}

	for _, path := range m.paths.CertificateKeyPaths {
		m.writeFile(path, secret.Data["tls.key"])
	}
}

func (m *Manager) writeFile(path string, data []byte) {
	log := m.logger.WithFields(log.Fields{
		"path": path,
	})

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}

	existingData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("file does not exist, creating file")

			err = os.WriteFile(path, data, 0644)
			if err != nil {
				log.Fatal(err)
			}

			return
		}

		m.logger.Fatal(err)
	}

	if bytes.Equal(existingData, data) {
		return
	}

	log.Info("file contents changed, updating file")

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
