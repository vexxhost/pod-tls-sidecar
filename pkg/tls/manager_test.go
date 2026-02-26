// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmfake "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kwatchpkg "k8s.io/apimachinery/pkg/watch"
	clientfeatures "k8s.io/client-go/features"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/rest"

	"github.com/vexxhost/pod-tls-sidecar/pkg/template"
)

// testFeatureGates disables WatchListClient so that the fake Kubernetes client
// works correctly with the informer without requiring bookmark events.
type testFeatureGates struct{}

func (testFeatureGates) Enabled(key clientfeatures.Feature) bool {
	if key == clientfeatures.WatchListClient {
		return false
	}
	return true
}

// TestMain replaces the global feature gates before any test is run so that
// the reflector does not use the WatchList mechanism (which requires bookmark
// events that the fake watch client does not produce).
func TestMain(m *testing.M) {
	clientfeatures.ReplaceFeatureGates(testFeatureGates{})
	os.Exit(m.Run())
}

const (
	testCertName  = "test-cert"
	testNamespace = "default"
)

// mockResolver implements net.Resolver for tests.
type mockResolver struct {
	hostname    string
	hostnameErr error
	fqdn        string
	fqdnErr     error
}

func (m *mockResolver) Hostname() (string, error) { return m.hostname, m.hostnameErr }
func (m *mockResolver) FQDN() (string, error)     { return m.fqdn, m.fqdnErr }

// newTestManager builds a Manager with fake clients injected via Config,
// bypassing NewManager to avoid the LoadValues / REST-config dependency.
func newTestManager(t *testing.T, cert *cmv1.Certificate, secret *v1.Secret, paths *WritePathConfig) (*Manager, *cmfake.Clientset, *k8sfake.Clientset) {
	t.Helper()

	objs := []runtime.Object{}
	if cert != nil {
		objs = append(objs, cert)
	}
	fakeCM := cmfake.NewSimpleClientset(objs...)

	k8sObjs := []runtime.Object{}
	if secret != nil {
		k8sObjs = append(k8sObjs, secret)
	}
	fakeK8s := k8sfake.NewSimpleClientset(k8sObjs...)

	if paths == nil {
		paths = &WritePathConfig{}
	}

	mgr := &Manager{
		config: &Config{
			Paths:    paths,
			OnUpdate: func() {},
		},
		certificate:       cert,
		certificateClient: fakeCM.CertmanagerV1().Certificates(testNamespace),
		logger:            log.WithFields(log.Fields{}),
		secretClient:      fakeK8s.CoreV1().Secrets(testNamespace),
	}
	return mgr, fakeCM, fakeK8s
}

// readyCert builds a Certificate with the Ready=True condition.
func readyCert() *cmv1.Certificate {
	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
		Status: cmv1.CertificateStatus{
			Conditions: []cmv1.CertificateCondition{
				{Type: cmv1.CertificateConditionReady, Status: cmmeta.ConditionTrue},
			},
		},
	}
}

// testSecret builds a bare Secret used by Create tests.
func testSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
	}
}

// interceptFatal makes logrus Fatal calls panic instead of calling os.Exit so
// that we can test the fatal-path code. The returned restore function must be
// called (usually via defer) to put back the original behaviour.
func interceptFatal() func() {
	log.StandardLogger().ExitFunc = func(int) { panic("log.Fatal") }
	return func() { log.StandardLogger().ExitFunc = nil }
}

// ─── NewManager ──────────────────────────────────────────────────────────────

func TestNewManager_LoadValuesError(t *testing.T) {
	// Unset required POD_* vars so podinfo.Load (called inside LoadValues) fails.
	keys := []string{"POD_UID", "POD_NAME", "POD_NAMESPACE", "POD_IP"}
	saved := make(map[string]string, len(keys))
	had := make(map[string]bool, len(keys))
	for _, k := range keys {
		saved[k], had[k] = os.LookupEnv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if had[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	})

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test
  namespace: default
spec:
  secretName: test
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithRestConfig(&rest.Config{Host: "http://localhost:8080"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)

	// Resolver left nil to exercise the default SystemResolver path.
	_, err = NewManager(config)
	require.Error(t, err)
}

func TestNewManager_HostnameResolverError(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test
  namespace: default
spec:
  secretName: test
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithRestConfig(&rest.Config{Host: "http://localhost:8080"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostnameErr: fmt.Errorf("hostname error")}

	_, err = NewManager(config)
	require.Error(t, err)
}

func TestNewManager_ExecuteError(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	// This template causes Execute to fail because len() on a struct is invalid.
	tmpl, err := template.New(`{{ len .PodInfo }}`)
	require.NoError(t, err)

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithRestConfig(&rest.Config{Host: "http://localhost:8080"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostname: "host", fqdn: "host.example.com"}

	_, err = NewManager(config)
	require.Error(t, err)
}

func TestNewManager_KubernetesClientError(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: "{{ .PodInfo.Name }}-ssl"
  namespace: "{{ .PodInfo.Namespace }}"
spec:
  secretName: "{{ .PodInfo.Name }}-ssl"
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	config, err := NewConfig(
		WithTemplate(tmpl),
		// Invalid URL makes kubernetes.NewForConfig return an error.
		WithRestConfig(&rest.Config{Host: "://invalid-url"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostname: "host", fqdn: "host.example.com"}

	_, err = NewManager(config)
	require.Error(t, err)
}

func TestNewManager_CMClientError(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: "{{ .PodInfo.Name }}-ssl"
  namespace: "{{ .PodInfo.Namespace }}"
spec:
  secretName: "{{ .PodInfo.Name }}-ssl"
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	// Pre-inject a valid SecretClient so the k8s client path is skipped,
	// but leave CertificateClient nil with an invalid RestConfig so the
	// cert-manager client creation fails.
	fakeK8s := k8sfake.NewSimpleClientset()

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithRestConfig(&rest.Config{Host: "://invalid-url"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostname: "host", fqdn: "host.example.com"}
	config.SecretClient = fakeK8s.CoreV1().Secrets("ns")

	_, err = NewManager(config)
	require.Error(t, err)
}

func TestNewManager_Success(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: "{{ .PodInfo.Name }}-ssl"
  namespace: "{{ .PodInfo.Namespace }}"
spec:
  secretName: "{{ .PodInfo.Name }}-ssl"
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithRestConfig(&rest.Config{Host: "http://localhost:8080"}),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostname: "host", fqdn: "host.example.com"}

	mgr, err := NewManager(config)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
}

func TestNewManager_SuccessWithInjectedClients(t *testing.T) {
	t.Setenv("POD_UID", "uid")
	t.Setenv("POD_NAME", "name")
	t.Setenv("POD_NAMESPACE", "ns")
	t.Setenv("POD_IP", "1.2.3.4")

	tmpl, err := template.New(`apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: "{{ .PodInfo.Name }}-ssl"
  namespace: "{{ .PodInfo.Namespace }}"
spec:
  secretName: "{{ .PodInfo.Name }}-ssl"
  issuerRef:
    kind: ClusterIssuer
    name: test
`)
	require.NoError(t, err)

	fakeK8s := k8sfake.NewSimpleClientset()
	fakeCM := cmfake.NewSimpleClientset()

	config, err := NewConfig(
		WithTemplate(tmpl),
		WithPaths(&WritePathConfig{}),
	)
	require.NoError(t, err)
	config.Resolver = &mockResolver{hostname: "host", fqdn: "host.example.com"}
	config.SecretClient = fakeK8s.CoreV1().Secrets("ns")
	config.CertificateClient = fakeCM.CertmanagerV1().Certificates("ns")

	mgr, err := NewManager(config)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	// RestConfig is nil, but no error since clients were injected.
	assert.Nil(t, config.RestConfig)
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	cert := readyCert()
	mgr, _, _ := newTestManager(t, cert, testSecret(), nil)

	err := mgr.Create(context.Background())
	require.NoError(t, err)
}

func TestCreate_AlreadyExists(t *testing.T) {
	cert := readyCert()
	mgr, _, _ := newTestManager(t, cert, testSecret(), nil)

	err := mgr.Create(context.Background())
	require.NoError(t, err)
}

func TestCreate_CreateError(t *testing.T) {
	mgr, fakeCM, _ := newTestManager(t, nil, nil, nil)
	mgr.certificate = &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
	}

	fakeCM.PrependReactor("create", "certificates",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("create error")
		})

	err := mgr.Create(context.Background())
	require.Error(t, err)
}

func TestCreate_GetError(t *testing.T) {
	mgr, fakeCM, _ := newTestManager(t, nil, nil, nil)
	mgr.certificate = &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
	}

	fakeCM.PrependReactor("get", "certificates",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("get error")
		})

	err := mgr.Create(context.Background())
	require.Error(t, err)
}

func TestCreate_CertNotReadyThenTimeout(t *testing.T) {
	cert := &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
		Status: cmv1.CertificateStatus{
			Conditions: []cmv1.CertificateCondition{
				{
					Type:    cmv1.CertificateConditionReady,
					Status:  cmmeta.ConditionFalse,
					Reason:  "Pending",
					Message: "waiting for CA",
				},
			},
		},
	}
	mgr, _, _ := newTestManager(t, cert, testSecret(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mgr.Create(ctx)
	require.Error(t, err) // context deadline exceeded
}

func TestCreate_PatchError(t *testing.T) {
	cert := readyCert()
	mgr, _, fakeK8s := newTestManager(t, cert, testSecret(), nil)

	fakeK8s.PrependReactor("patch", "secrets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("patch error")
		})

	err := mgr.Create(context.Background())
	require.Error(t, err)
}

// ─── write ───────────────────────────────────────────────────────────────────

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()
	caPath := filepath.Join(tmpDir, "ca.crt")
	certPath := filepath.Join(tmpDir, "tls.crt")
	keyPath := filepath.Join(tmpDir, "tls.key")

	onUpdateCalled := false
	mgr := &Manager{
		config: &Config{
			Paths: &WritePathConfig{
				CertificateAuthorityPaths: []string{caPath},
				CertificatePaths:          []string{certPath},
				CertificateKeyPaths:       []string{keyPath},
			},
			OnUpdate: func() { onUpdateCalled = true },
		},
		certificate: &cmv1.Certificate{},
		logger:      log.WithFields(log.Fields{}),
	}

	secret := &v1.Secret{
		Data: map[string][]byte{
			"ca.crt":  []byte("ca-data"),
			"tls.crt": []byte("cert-data"),
			"tls.key": []byte("key-data"),
		},
	}

	mgr.write(secret)

	assert.True(t, onUpdateCalled)

	data, err := os.ReadFile(caPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("ca-data"), data)

	data, err = os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("cert-data"), data)

	data, err = os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("key-data"), data)
}

// ─── writeFile ───────────────────────────────────────────────────────────────

func newWriteFileMgr() *Manager {
	return &Manager{
		config:      &Config{Paths: &WritePathConfig{}, OnUpdate: func() {}},
		certificate: &cmv1.Certificate{},
		logger:      log.WithFields(log.Fields{}),
	}
}

func TestWriteFile_NewFile(t *testing.T) {
	mgr := newWriteFileMgr()
	path := filepath.Join(t.TempDir(), "test.txt")
	data := []byte("hello")

	mgr.writeFile(path, data)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestWriteFile_SameContent(t *testing.T) {
	mgr := newWriteFileMgr()
	path := filepath.Join(t.TempDir(), "test.txt")
	data := []byte("hello")

	require.NoError(t, os.WriteFile(path, data, 0644))

	mgr.writeFile(path, data)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestWriteFile_UpdatedContent(t *testing.T) {
	mgr := newWriteFileMgr()
	path := filepath.Join(t.TempDir(), "test.txt")

	require.NoError(t, os.WriteFile(path, []byte("old"), 0644))

	newData := []byte("new")
	mgr.writeFile(path, newData)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, newData, got)
}

func TestWriteFile_NestedDirCreation(t *testing.T) {
	mgr := newWriteFileMgr()
	path := filepath.Join(t.TempDir(), "a", "b", "c", "test.txt")

	mgr.writeFile(path, []byte("nested"))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("nested"), got)
}

func TestWriteFile_FatalOnMkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: test requires non-root execution")
	}

	mgr := newWriteFileMgr()

	blockingFile := filepath.Join(t.TempDir(), "parent")
	require.NoError(t, os.WriteFile(blockingFile, []byte("I am a file"), 0644))
	path := filepath.Join(blockingFile, "child.txt")

	defer interceptFatal()()
	assert.Panics(t, func() { mgr.writeFile(path, []byte("data")) })
}

func TestWriteFile_FatalOnWriteNewFileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: test requires non-root execution")
	}

	mgr := newWriteFileMgr()

	readOnlyDir := filepath.Join(t.TempDir(), "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0555))
	path := filepath.Join(readOnlyDir, "file.txt")

	defer interceptFatal()()
	assert.Panics(t, func() { mgr.writeFile(path, []byte("data")) })
}

func TestWriteFile_FatalOnReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: test requires non-root execution")
	}

	mgr := newWriteFileMgr()

	path := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0644))
	require.NoError(t, os.Chmod(path, 0000))
	defer os.Chmod(path, 0644) //nolint:errcheck

	defer interceptFatal()()
	assert.Panics(t, func() { mgr.writeFile(path, []byte("new")) })
}

func TestWriteFile_FatalOnUpdateError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: test requires non-root execution")
	}

	mgr := newWriteFileMgr()

	path := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0644))
	require.NoError(t, os.Chmod(path, 0444))
	defer os.Chmod(path, 0644) //nolint:errcheck

	defer interceptFatal()()
	assert.Panics(t, func() { mgr.writeFile(path, []byte("updated")) })
}

// ─── watch (informer loop) ────────────────────────────────────────────────────

// buildWatchMgr returns a Manager wired to a fake k8s clientset that has
// 'secret' pre-seeded. It also registers a watch reactor that stores the
// FakeWatcher so tests can inject events.
func buildWatchMgr(t *testing.T, secret *v1.Secret, paths *WritePathConfig) (*Manager, *k8sfake.Clientset, **kwatchpkg.FakeWatcher, chan struct{}) {
	t.Helper()

	fakeK8s := k8sfake.NewSimpleClientset(secret)

	var fw *kwatchpkg.FakeWatcher
	watcherReady := make(chan struct{})

	fakeK8s.PrependWatchReactor("secrets",
		func(action k8stesting.Action) (bool, kwatchpkg.Interface, error) {
			fw = kwatchpkg.NewFake()
			close(watcherReady)
			return true, fw, nil
		})

	if paths == nil {
		paths = &WritePathConfig{}
	}

	onUpdateCalled := make(chan struct{}, 10)
	mgr := &Manager{
		config: &Config{
			Paths: paths,
			OnUpdate: func() {
				select {
				case onUpdateCalled <- struct{}{}:
				default:
				}
			},
			WatchRetryDelay: 10 * time.Millisecond,
		},
		certificate: &cmv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
		},
		logger:       log.WithFields(log.Fields{}),
		secretClient: fakeK8s.CoreV1().Secrets(testNamespace),
	}

	return mgr, fakeK8s, &fw, onUpdateCalled
}

func TestWatch_AddFunc(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
		Data: map[string][]byte{
			"ca.crt":  []byte("ca"),
			"tls.crt": []byte("cert"),
			"tls.key": []byte("key"),
		},
	}

	tmpDir := t.TempDir()
	paths := &WritePathConfig{
		CertificateAuthorityPaths: []string{filepath.Join(tmpDir, "ca.crt")},
		CertificatePaths:          []string{filepath.Join(tmpDir, "tls.crt")},
		CertificateKeyPaths:       []string{filepath.Join(tmpDir, "tls.key")},
	}

	mgr, _, _, onUpdateCalled := buildWatchMgr(t, secret, paths)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.watch(ctx)

	select {
	case <-onUpdateCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("AddFunc not called within timeout")
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "ca.crt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("ca"), data)

	cancel()
}

func TestWatch_UpdateFunc(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
		Data:       map[string][]byte{"ca.crt": []byte("original")},
	}

	tmpDir := t.TempDir()
	paths := &WritePathConfig{
		CertificateAuthorityPaths: []string{filepath.Join(tmpDir, "ca.crt")},
	}

	mgr, _, fwPtr, onUpdateCalled := buildWatchMgr(t, secret, paths)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.watch(ctx)

	select {
	case <-onUpdateCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("AddFunc not called")
	}

	updated := secret.DeepCopy()
	updated.Data = map[string][]byte{"ca.crt": []byte("updated")}
	(*fwPtr).Modify(updated)

	select {
	case <-onUpdateCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("UpdateFunc not called")
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "ca.crt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), data)

	cancel()
}

func TestWatch_DeleteFunc(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
	}

	mgr, _, fwPtr, onUpdateCalled := buildWatchMgr(t, secret, nil)

	fatalCalled := make(chan struct{}, 1)
	log.StandardLogger().ExitFunc = func(int) {
		select {
		case fatalCalled <- struct{}{}:
		default:
		}
	}
	defer func() { log.StandardLogger().ExitFunc = nil }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.watch(ctx)

	select {
	case <-onUpdateCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("AddFunc not called")
	}

	(*fwPtr).Delete(secret)

	select {
	case <-fatalCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("Fatal not called on secret deletion")
	}

	cancel()
}

// ─── Watch (retry loop) ───────────────────────────────────────────────────────

func TestWatch_RetryLoop(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testCertName, Namespace: testNamespace},
	}

	mgr, _, _, onUpdateCalled := buildWatchMgr(t, secret, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		mgr.Watch(ctx)
	}()

	// Wait for AddFunc to confirm the informer loop ran.
	select {
	case <-onUpdateCalled:
	case <-time.After(10 * time.Second):
		t.Fatal("Watch/watch did not call AddFunc")
	}

	// Allow the retry loop to fire (the informer returns, triggering the
	// retry delay), then cancel on the second iteration.
	// The WatchRetryDelay is 10ms (set by buildWatchMgr), so this resolves
	// quickly. We wait a bit to let the retry log message and delay execute.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not stop after context cancellation")
	}
}

func TestWatchRetryDelay_Default(t *testing.T) {
	mgr := &Manager{
		config: &Config{},
	}
	assert.Equal(t, defaultWatchRetryDelay, mgr.watchRetryDelay())
}

func TestWatchRetryDelay_Custom(t *testing.T) {
	mgr := &Manager{
		config: &Config{WatchRetryDelay: 42 * time.Second},
	}
	assert.Equal(t, 42*time.Second, mgr.watchRetryDelay())
}
