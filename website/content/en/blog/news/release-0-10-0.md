---
title: "v0.10.0"
linkTitle: "v0.10.0"
slug: "release-0-10-0"
date: 2026-11-14
draft: false
description: >
  Jenkins Operator 0.10.0 adds cloud identity and service mesh support, ephemeral PVCs, and namespace fixes
---

Jenkins Operator **0.10.0** adds service mesh and cloud identity support, ephemeral storage, and more control over agent scheduling.

- **Istio and Linkerd:** Jenkins Operator now supports Istio and Linkerd service meshes. For Istio, set `sidecar.istio.io/rewriteAppHTTPProbers: "false"`; full strict-mTLS support is not included.
- **Cloud identity:** Support for AWS IRSA, EKS Pod Identity, Azure Workload Identity and Google ADC injection now prevents injected fields from causing Jenkins Pod recreation. Custom rules are available for other injected fields.
- **Ephemeral storage:** You can now use an ephemeral PVC for Jenkins home, including a custom StorageClass. `emptyDir` remains the default.
- **Seed-job agent:** The agent is now reserved for seed jobs by default. To use it for other jobs, set `jenkins.seedJobRestrictJobsToLabel=false` in Helm or `spec.seedJobRestrictJobsToLabel: false` in the Jenkins resource.
- **Jenkins telemetry:** At the Jenkins Foundation's request, the operator no longer disables telemetry by default. To opt out, set `jenkins.disableUsageStatistics=true` in Helm or `spec.master.disableUsageStatistics: true` in the Jenkins resource.
- **Operator watch namespace:** The chart now creates RBAC in the namespace the operator actually watches. Standalone installations watch their own namespace by default; use `operator.watchNamespace` for another namespace or `operator.watchAllNamespaces=true` for all namespaces.
- **Custom Jenkins images:** You can now set the Jenkins image repository and tag separately with `jenkins.image.repository` and `jenkins.image.tag`. Existing image strings remain supported.
