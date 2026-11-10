---
title: "Jenkins Docker Images"
linkTitle: "Jenkins Docker Images"
weight: 10
date: 2026-10-31
description: >
  Jenkins default image details
---

Jenkins Operator uses the official `jenkins/jenkins` image. Chart 0.9.0 pins
`jenkins/jenkins:2.580.1-lts` rather than the moving `lts` tag so deployments use a known version.

You can choose another image through `jenkins.image` in the chart or
`spec.master.containers` in the Jenkins resource. A custom image must retain the standard Jenkins
filesystem layout and provide the configured probe endpoints. The 0.9.0 defaults use `/health`;
explicit custom probe paths are preserved, including the configured Jenkins URL prefix.

To use an image with preinstalled plugins, set chart value `jenkins.skipPlugins=true`, or
`spec.master.skipPlugins: true` in the resource. This skips installation of both base and additional
plugins; the image must already include the required base plugins.

The operator and PVC backup images are published separately on Quay. Release 0.9.0 uses
operator `v0.9.0` and backup `v0.5.1`; both support Linux AMD64 and ARM64.
