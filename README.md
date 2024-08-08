# `pod-tls-sidecar`

This projects aims to provide a simple sidecar container that can be used to
issue TLS certificates to pods in a Kubernetes cluster using [cert-manager](https://cert-manager.io/).

## FAQ

### Why not use `cert-manager` directly?

The goal of this project is to allow you to dynamically issue certificates to pods
based on the pod's identity.  This is useful in cases where you want to issue
certificates for mTLS between pods, or when you need to control the certificate
fields based on the pod's identity.
