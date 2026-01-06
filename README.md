# `pod-tls-sidecar`

This projects aims to provide a simple sidecar container that can be used to
issue TLS certificates to pods in a Kubernetes cluster using [cert-manager](https://cert-manager.io/).

## FAQ

### Why not use `cert-manager` directly?

The goal of this project is to allow you to dynamically issue certificates to pods
based on the pod's identity.  This is useful in cases where you want to issue
certificates for mTLS between pods, or when you need to control the certificate
fields based on the pod's identity.

## Creating a Release

This project uses [semantic versioning](https://semver.org/) with version tags
prefixed with `v` (e.g., `v1.0.0`, `v1.2.3`).

To create a new release:

1. Go to the [Releases](https://github.com/vexxhost/pod-tls-sidecar/releases) page on GitHub
2. Click **Draft a new release**
3. Click **Choose a tag** and type your new version (e.g., `v1.0.0`)
4. Select **Create new tag: vX.Y.Z on publish**
5. Set the **Target** to `main` (or appropriate branch)
6. Enter a **Release title** (typically the version number, e.g., `v1.0.0`)
7. Click **Generate release notes** to auto-generate the changelog from merged
   pull requests and commits since the last release
8. Review and edit the generated notes if needed
9. Click **Publish release**

### Version Numbering

Follow semantic versioning (`MAJOR.MINOR.PATCH`):

- **MAJOR**: Breaking changes or incompatible API changes
- **MINOR**: New features that are backwards compatible
- **PATCH**: Backwards compatible bug fixes
