---
title: "AKS"
linkTitle: "AKS"
weight: 8
date: 2021-12-08
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

## Azure Workload Identity (unreleased reconciliation options)

These options require an operator image and CRD built after 0.9.0; the published
0.9.0 release does not support them. Configure the ServiceAccount and opt-in label,
with the built-in injection rules enabled by default:

```yaml
spec:
  serviceAccount:
    annotations:
      azure.workload.identity/client-id: YOUR_CLIENT_ID
  master:
    labels:
      azure.workload.identity/use: "true"
  reconciliation:
    ignoreWellKnownInjections: true
```

This covers Azure's fixed and generated token-volume names and injected envs.
See the [built-in ignore list](../customizing-jenkins/#built-in-cloud-identity-rules)
for exact names and scope. Add manual rules only for other injected fields.
