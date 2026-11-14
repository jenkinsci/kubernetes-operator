---
title: "Installing the Operator"
linkTitle: "Installing the Operator"
weight: 1
date: 2026-10-31
description: >
  Install Jenkins Operator 0.9.0 with Helm or release manifests
---

{{% pageinfo %}}
This guide installs **Jenkins Operator 0.9.0**. Official images are published on Quay:
[operator](https://quay.io/repository/jenkins-kubernetes-operator/operator) and
[backup-pvc](https://quay.io/repository/jenkins-kubernetes-operator/backup-pvc).
Both release images support Linux AMD64 and ARM64.
{{% /pageinfo %}}

## Requirements

- A Kubernetes cluster and permission to create the operator's CRD and RBAC resources.
- `kubectl` configured for that cluster, using a version compatible with its API server.
- Helm 3 if installing the chart.

See the [release notes](https://github.com/jenkinsci/kubernetes-operator/releases/tag/v0.9.0)
and [release CI](https://github.com/jenkinsci/kubernetes-operator/actions/workflows/auto-tests-e2e.yaml)
when evaluating your cluster version.

## Install using Helm

Add the chart repository and install chart version 0.9.0:

```bash
helm repo add jenkins https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart
helm repo update
helm install jenkins-operator jenkins/jenkins-operator \
  --version 0.9.0 --namespace jenkins --create-namespace \
  --set jenkins.namespace=jenkins
```

By default, this installs both the operator and a Jenkins instance. To install only the operator,
add `--set jenkins.enabled=false`, then follow [Deploying Jenkins](/kubernetes-operator/docs/getting-started/v0.9.x/deploying-jenkins/).

Chart 0.9.0 uses these defaults:

| Component | Image |
|---|---|
| Operator | `quay.io/jenkins-kubernetes-operator/operator:v0.9.0` |
| Jenkins | `jenkins/jenkins:2.580.1-lts` |
| Backup sidecar | `quay.io/jenkins-kubernetes-operator/backup-pvc:v0.5.1` |

The chart version, operator version, Jenkins version, and backup version are independent.
Use a custom values file with `-f values.yaml` to override settings. See the
[0.9.0 chart reference](https://github.com/jenkinsci/kubernetes-operator/blob/v0.9.0/chart/jenkins-operator/README.md)
for supported values, including `jenkins.lifecycle`, `jenkins.terminationGracePeriodSeconds`,
`jenkins.skipPlugins`, and `operator.watchNamespace`.

For example, to add Jenkins labels and annotations:

```bash
helm install jenkins-operator jenkins/jenkins-operator \
  --version 0.9.0 --namespace jenkins --create-namespace \
  --set jenkins.namespace=jenkins \
  --set jenkins.labels.LabelKey=LabelValue \
  --set jenkins.annotations.AnnotationKey=AnnotationValue
```

## Install using release manifests

Install the CRD, then the operator and its RBAC resources:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/v0.9.0/config/crd/bases/jenkins.io_jenkins.yaml
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/v0.9.0/deploy/all-in-one-v1alpha2.yaml
kubectl rollout status deployment/jenkins-operator --namespace default
```

This installs the operator in `default`; it does not create a Jenkins instance.
Continue with [Deploying Jenkins](/kubernetes-operator/docs/getting-started/v0.9.x/deploying-jenkins/).
For separate operator and Jenkins namespaces, see the
[separate namespaces guide](/kubernetes-operator/docs/getting-started/v0.9.x/separate-namespaces/).

## Upgrade an existing installation

Back up Jenkins data and save your custom values before upgrading. Review the
[release notes](https://github.com/jenkinsci/kubernetes-operator/releases/tag/v0.9.0)
and [backup storage requirements](/kubernetes-operator/docs/getting-started/v0.9.x/configuring-backup-and-restore/#pvc-storage-size).

Helm does not upgrade CRDs from a chart's `crds/` directory. Apply the release CRD before
upgrading the chart, using your existing release name, namespace, and custom values:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/v0.9.0/config/crd/bases/jenkins.io_jenkins.yaml
helm repo update
helm upgrade jenkins-operator jenkins/jenkins-operator \
  --version 0.9.0 --namespace jenkins -f values.yaml
```

Check the rendered values before applying them, particularly custom images, probes, backup settings,
and any use of the new `skipPlugins` option. The CR API remains `jenkins.io/v1alpha2`.

## Validating webhook

The webhook can reject Jenkins resources whose configured base or additional plugin versions match
security warnings from Jenkins Update Center metadata. This validation is opt-in: enable both the
operator's webhook and `spec.validateSecurityWarnings` on the Jenkins resource.

The webhook requires cert-manager for its serving certificate. If cert-manager is already installed,
install the operator first with Jenkins creation disabled:

```bash
helm upgrade --install jenkins-operator jenkins/jenkins-operator \
  --version 0.9.0 --namespace jenkins --create-namespace \
  --set jenkins.enabled=false --set webhook.enabled=true
```

If cert-manager is not installed, the chart can install its bundled cert-manager dependency and CRDs
with `--set cert-manager.enabled=true`. Check that dependency's compatibility with your cluster.
Wait for cert-manager and the operator's serving certificate before creating a Jenkins resource.

For a Helm-managed Jenkins instance, set `jenkins.enabled=true` and
`jenkins.validateSecurityWarnings=true` in your values file once the webhook is ready.
For a resource managed separately, set:

```yaml
spec:
  validateSecurityWarnings: true
```

The operator needs access to `https://updates.jenkins.io/update-center.actual.json` to fetch warning metadata.
Validation rejects a resource if that metadata has not yet been fetched. See
[Security](/kubernetes-operator/docs/security/) for the validation behavior and other hardening settings.

## Development snapshots

Nightly images are published to the [operator repository](https://quay.io/repository/jenkins-kubernetes-operator/operator)
with a short Git commit hash as their tag. These are development snapshots, not the stable 0.9.0 image.
Use the `master` manifests only when intentionally testing unreleased changes.

## Note on restricted Jenkins controller pod volumeMounts
Current design of the Operator puts an emphasis on creating a full GitOps flow of work for Jenkins users.
One of the key points of this design is maintaining an immutable state of Jenkins.

One of the prerequisites of this is an ephemeral Jenkins home directory. To achieve that, Operator mounts emptyDir Volume
(jenkins-home) as Jenkins home directory.
It is not possible to overwrite volumeMount and specify any other Volume for Jenkins home directory,
as attempting to do so will result in Operator error.

jenkins-home is not the only Jenkins controller pod volumeMount that is non-configurable and managed by Operator,
below is the full list of those volumeMounts:

* jenkins-home
* scripts
* init-configuration
* operator-credentials
