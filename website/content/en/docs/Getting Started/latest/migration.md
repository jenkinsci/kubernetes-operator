---
title: "Migration from v0.2.x"
linkTitle: "Migration from v0.2.x"
weight: 10
date: 2019-08-05
description: >
    How to migrate from v0.2.x to v0.3.x
---

### Changes

- new Jenkins Custom Resource Definition version `jenkins.io/v1alpha2`:
  - `spec.master.masterAnnotations` was deprecated, use `spec.master.masterAnnotations`
  - added `spec.notifications`
  - added `spec.master.tolerations` (in v0.3.1)
  - added `spec.master.disableCSRFProtection`

### Migration

- adjust the operator image version, e.g. `image: virtuslab/jenkins-operator:v0.3.1`
- migrate your Jenkins Custom Resources to `apiVersion: jenkins.io/v1alpha2`, adjust content if necessary

The v0.3.x should work fine with `jenkins.io/v1alpha1`, but we recommend using `jenkins.io/v1alpha2`.
