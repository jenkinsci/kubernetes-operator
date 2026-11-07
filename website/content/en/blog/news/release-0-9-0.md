---
title: "v0.9.0"
linkTitle: "v0.9.0"
slug: "release-0-9-0"
date: 2026-10-31
description: >
  Jenkins Operator 0.9.0 brings ARM64 backups, updated Jenkins defaults, and improved reliability
---

## New in this Release

Jenkins Operator **0.9.0** is here! 🎉 This release brings together the fixes since 0.8.0,
including 0.8.1 and both 0.9.0 betas. Thanks to everyone who contributed and helped test it! 🙏

### Jenkins

- Updated the default Jenkins image to **2.580.1 LTS** and refreshed base plugins.
- Updated the seed-job agent to Java 21 and WebSocket connections.
- Added `skipPlugins` for images with preinstalled plugins, lifecycle hooks, and configurable shutdown grace periods.
- Fixed ConfigMap watches and improved health probes and plugin-security validation.

### Backup

- Published **AMD64 and ARM64** backup images; the chart now uses **v0.5.1**, based on Debian Trixie.
- Improved backup/restore locking, atomic backup creation, retention handling, and logs.
- Added support for arbitrary container user IDs for OpenShift compatibility.

### CI / Charts

- Added backup tests for both architectures and publish both image tags from one build.
- Updated to **Go 1.26** and improved Nix tooling, integration tests, and Helm configuration.

Back up Jenkins data and review the [installation and upgrade guide](/kubernetes-operator/docs/getting-started/latest/installing-the-operator/)
before updating. Apply the release CRD first, then upgrade your existing Helm release with your custom values:

```bash
helm upgrade jenkins-operator jenkins/jenkins-operator --version 0.9.0 --namespace jenkins -f values.yaml
```

Ensure the [backup PVC has room for temporary backups](/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/#pvc-storage-size).
The CR API remains `jenkins.io/v1alpha2`.

See the [release notes](https://github.com/jenkinsci/kubernetes-operator/releases/tag/v0.9.0)
and [full changelog](https://github.com/jenkinsci/kubernetes-operator/compare/v0.8.0...v0.9.0) for details.
