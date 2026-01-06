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

To create a new release using the [GitHub CLI](https://cli.github.com/):

```bash
gh release create vX.Y.Z --generate-notes
```

This will:

- Create a new tag `vX.Y.Z` on the current branch
- Auto-generate release notes from merged pull requests and commits since the
  last release
- Publish the release to GitHub

### Version Numbering

Follow semantic versioning (`MAJOR.MINOR.PATCH`):

- **MAJOR**: Breaking changes or incompatible API changes
- **MINOR**: New features that are backwards compatible
- **PATCH**: Backwards compatible bug fixes
