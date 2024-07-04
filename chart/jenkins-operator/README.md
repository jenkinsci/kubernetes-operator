# jenkins-operator

![Version: 0.8.0](https://img.shields.io/badge/Version-0.8.0-informational?style=flat-square) ![AppVersion: 0.8.0](https://img.shields.io/badge/AppVersion-0.8.0-informational?style=flat-square)

Kubernetes native operator which fully manages Jenkins on Kubernetes

## Requirements

| Repository | Name | Version |
|------------|------|---------|
|  | cert-manager-crds | 1.14.2 |
| https://charts.jetstack.io | cert-manager | 1.14.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cert-manager.enabled | bool | `false` |  |
| cert-manager.startupapicheck.enabled | bool | `false` |  |
| jenkins.annotations | object | `{}` |  |
| jenkins.apiVersion | string | `"jenkins.io/v1alpha2"` |  |
| jenkins.authorizationStrategy | string | `"createUser"` |  |
| jenkins.backup.backupCommand[0] | string | `"/home/user/bin/backup.sh"` |  |
| jenkins.backup.containerName | string | `"backup"` |  |
| jenkins.backup.enabled | bool | `true` |  |
| jenkins.backup.env[0].name | string | `"BACKUP_DIR"` |  |
| jenkins.backup.env[0].value | string | `"/backup"` |  |
| jenkins.backup.env[1].name | string | `"JENKINS_HOME"` |  |
| jenkins.backup.env[1].value | string | `"/jenkins-home"` |  |
| jenkins.backup.env[2].name | string | `"BACKUP_COUNT"` |  |
| jenkins.backup.env[2].value | string | `"3"` |  |
| jenkins.backup.getLatestAction[0] | string | `"/home/user/bin/get-latest.sh"` |  |
| jenkins.backup.image | string | `"quay.io/jenkins-kubernetes-operator/backup-pvc:v0.4.1"` |  |
| jenkins.backup.interval | int | `30` |  |
| jenkins.backup.makeBackupBeforePodDeletion | bool | `true` |  |
| jenkins.backup.pvc.className | string | `""` |  |
| jenkins.backup.pvc.enabled | bool | `true` |  |
| jenkins.backup.pvc.size | string | `"5Gi"` |  |
| jenkins.backup.resources.limits.cpu | string | `"1000m"` |  |
| jenkins.backup.resources.limits.memory | string | `"2Gi"` |  |
| jenkins.backup.resources.requests.cpu | string | `"100m"` |  |
| jenkins.backup.resources.requests.memory | string | `"500Mi"` |  |
| jenkins.backup.restoreCommand[0] | string | `"/home/user/bin/restore.sh"` |  |
| jenkins.backup.volumeMounts[0].mountPath | string | `"/jenkins-home"` |  |
| jenkins.backup.volumeMounts[0].name | string | `"jenkins-home"` |  |
| jenkins.backup.volumeMounts[1].mountPath | string | `"/backup"` |  |
| jenkins.backup.volumeMounts[1].name | string | `"backup"` |  |
| jenkins.basePlugins | list | `[]` |  |
| jenkins.configuration.configurationAsCode | list | `[]` |  |
| jenkins.configuration.groovyScripts | list | `[]` |  |
| jenkins.configuration.secretData | object | `{}` |  |
| jenkins.configuration.secretRefName | string | `""` |  |
| jenkins.disableCSRFProtection | bool | `false` |  |
| jenkins.enabled | bool | `true` |  |
| jenkins.env | list | `[]` |  |
| jenkins.hostAliases | object | `{}` |  |
| jenkins.image | string | `"jenkins/jenkins:2.452.2-lts"` |  |
| jenkins.imagePullPolicy | string | `"Always"` |  |
| jenkins.imagePullSecrets | list | `[]` |  |
| jenkins.labels | object | `{}` |  |
| jenkins.latestPlugins | bool | `true` |  |
| jenkins.livenessProbe.failureThreshold | int | `20` |  |
| jenkins.livenessProbe.httpGet.path | string | `"/login"` |  |
| jenkins.livenessProbe.httpGet.port | string | `"http"` |  |
| jenkins.livenessProbe.httpGet.scheme | string | `"HTTP"` |  |
| jenkins.livenessProbe.initialDelaySeconds | int | `100` |  |
| jenkins.livenessProbe.periodSeconds | int | `10` |  |
| jenkins.livenessProbe.successThreshold | int | `1` |  |
| jenkins.livenessProbe.timeoutSeconds | int | `8` |  |
| jenkins.name | string | `"jenkins"` |  |
| jenkins.namespace | string | `"default"` |  |
| jenkins.nodeSelector | object | `{}` |  |
| jenkins.notifications | list | `[]` |  |
| jenkins.plugins | list | `[]` |  |
| jenkins.priorityClassName | string | `""` |  |
| jenkins.readinessProbe.failureThreshold | int | `60` |  |
| jenkins.readinessProbe.httpGet.path | string | `"/login"` |  |
| jenkins.readinessProbe.httpGet.port | string | `"http"` |  |
| jenkins.readinessProbe.httpGet.scheme | string | `"HTTP"` |  |
| jenkins.readinessProbe.initialDelaySeconds | int | `120` |  |
| jenkins.readinessProbe.periodSeconds | int | `10` |  |
| jenkins.readinessProbe.successThreshold | int | `1` |  |
| jenkins.readinessProbe.timeoutSeconds | int | `8` |  |
| jenkins.resources.limits.cpu | string | `"1000m"` |  |
| jenkins.resources.limits.memory | string | `"3Gi"` |  |
| jenkins.resources.requests.cpu | string | `"250m"` |  |
| jenkins.resources.requests.memory | string | `"500Mi"` |  |
| jenkins.securityContext.fsGroup | int | `1000` |  |
| jenkins.securityContext.runAsUser | int | `1000` |  |
| jenkins.seedJobAgentImage | string | `""` |  |
| jenkins.seedJobs | list | `[]` |  |
| jenkins.serviceAccount.annotations | object | `{}` |  |
| jenkins.terminationGracePeriodSeconds | int | `30` |  |
| jenkins.tolerations | list | `[]` |  |
| jenkins.validateSecurityWarnings | bool | `false` |  |
| jenkins.volumeMounts | list | `[]` |  |
| jenkins.volumes[0].name | string | `"backup"` |  |
| jenkins.volumes[0].persistentVolumeClaim.claimName | string | `"jenkins-backup"` |  |
| operator.affinity | object | `{}` |  |
| operator.fullnameOverride | string | `""` |  |
| operator.image | string | `"quay.io/jenkins-kubernetes-operator/operator:v0.8.1"` |  |
| operator.imagePullPolicy | string | `"IfNotPresent"` |  |
| operator.imagePullSecrets | list | `[]` |  |
| operator.nameOverride | string | `""` |  |
| operator.nodeSelector | object | `{}` |  |
| operator.replicaCount | int | `1` |  |
| operator.resources | object | `{}` |  |
| operator.tolerations | list | `[]` |  |
| webhook.certificate.duration | string | `"2160h"` |  |
| webhook.certificate.name | string | `"webhook-certificate"` |  |
| webhook.certificate.renewbefore | string | `"360h"` |  |
| webhook.enabled | bool | `false` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.2](https://github.com/norwoodj/helm-docs/releases/v1.11.2)
