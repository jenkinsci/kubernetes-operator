---
title: "AKS"
linkTitle: "AKS"
weight: 10
date: 2021-01-18
description: >
    Additional configuration for Azure Kubernetes Service
---

Azure AKS managed Kubernetes service adds to every pod the following environment variables:

```yaml
- name: KUBERNETES_PORT_443_TCP_ADDR
  value:
- name: KUBERNETES_PORT
  value: tcp://
- name: KUBERNETES_PORT_443_TCP
  value: tcp://
- name: KUBERNETES_SERVICE_HOST
  value:
```

The operator is aware of it and omits these environment variables when checking if a Jenkins pod environment has been changed. It prevents the 
restart of a Jenkins pod over and over again.