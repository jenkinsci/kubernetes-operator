---
title: "v0.8.1"
linkTitle: "v0.8.1"
date: 2024-04-07
description: >
  v0.8.1 has been released
---

# New in this Release

## Jenkins

* Updated the base plugins and Jenkins LTS to the latest version.
* Exposed `terminationGracePeriodSeconds` for the Jenkins master.

## Backup

* From version `4.1`, the backup temporary directory is created in the backup PVC, so the temporary tar file is moved immediately after the tar process ends. This improves speed and reliability. Size your backup PVC accordingly. See the [documentation](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/#pvc-storage-size) section `PVC Storage Size`.
* From version `4.1`, each backup component (backup, restore, run) logs everything to stdout/stderr, improving understanding in case of issues and providing a backup and restore timeline. You can have a preview of the logs from the [PR](https://github.com/jenkinsci/kubernetes-operator/pull/1023) description.
* From version `4.1`, a lock file for backup and restore processes is implemented to prevent multiple backups in case of operator crash/restart. See this [section](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/#Customizing-pvc-backup-behaviour) of the docs.

If you do not wish to use these new features, you can use the old `v0.2.6` tag.

## CI / Chart

* Cert-manager CRDs are no longer shipped by default.
* Improved Ginkgo tests by implementing matrix testing.
* Enhanced BATS tests to cover more scenarios.


## Bug Fixes

* Fix incorrect double restore in the reconciliation loop
* Fixed `configurationAsCode` and `groovyScripts` arrays in the Helm chart.
* Ensured consistent `imagePullPolicy` for the backup container.
* Stopped using deprecated `jnlpUrl` for getting seed agents node secret.
* Fixed the website build CI.

Thanks to all contributors who made this release possible!
