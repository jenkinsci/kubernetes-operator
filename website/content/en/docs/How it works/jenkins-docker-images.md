---
title: "Jenkins Docker Images"
linkTitle: "Jenkins Docker Images"
weight: 10
date: 2019-08-05
description: >
  Jenkins default image details
---

**Jenkins Operator** is fully compatible with **`jenkins:lts`** Docker image and does not introduce any hidden changes 
to the upstream Jenkins. However due to problems with plugins and images version compatibility we are using specific tags 
in the exemplary Custom Resource, so you know a working configuration.

If needed, the Docker image can be easily changed in custom resource manifest as long as it supports standard Jenkins file system structure.
