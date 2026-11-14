---
title: "OpenShift"
linkTitle: "OpenShift"
weight: 10
date: 2026-11-02
description: >
    OpenShift image compatibility for Jenkins Operator 0.10.0
---

## Jenkins image compatibility

Chart 0.10.0 uses the official `jenkins/jenkins:2.580.2-lts` image. Compatibility with the
Red Hat-provided Jenkins image has not been verified for this release. The earlier report in
[issue #826](https://github.com/jenkinsci/kubernetes-operator/issues/826) concerns an older
operator version; it does not establish the compatibility of 0.10.0.

The PVC backup image supports execution with arbitrary user IDs, but that alone does not
verify the complete deployment against OpenShift security constraints. Test the chosen Jenkins
image and security context on your OpenShift cluster before upgrading. Please include the
operator version, image tags, OpenShift version, and relevant logs when reporting a problem.
