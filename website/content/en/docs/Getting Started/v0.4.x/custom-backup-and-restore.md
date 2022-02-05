---
title: "Custom Backup and Restore Providers"
linkTitle: "Custom Backup and Restore Providers"
weight: 10
date: 2020-04-13
description: >
  Custom backup and restore provider
---

With enough effort one can create a custom backup and restore provider 
for the Jenkins Operator.

## Requirements

Two commands (e.g. scripts) are required:

- a backup command, e.g. `backup.sh` that takes one argument, a **backup number**
- a restore command, e.g. `backup.sh` that takes one argument, a **backup number**

Both scripts need to return an exit code of `0` on success and `1` or greater for failure.

One of those scripts (or the entry point of the container) needs to be responsible
for backup cleanup or rotation if required, or an external system.

## How it works

The mechanism relies on basic Kubernetes and UNIX functionalities.

The backup (and restore) container runs as a sidecar in the same 
Kubernetes pod as the Jenkins master.

Name of the backup and restore containers can be set as necessary using 
`spec.backup.containerName` and `spec.restore.containerName`. 
In most cases it will be the same container, but we allow for less common use cases.

The operator will call a backup or restore commands inside a sidecar container when necessary:

- backup command (defined in `spec.backup.action.exec.command`) 
  will be called every `N` seconds configurable in: `spec.backup.interval`
  and on pod shutdown (if enabled in `spec.backup.makeBackupBeforePodDeletion`)
  with an integer representing the current backup number as first and only argument
- restore command (defined in `spec.restore.action.exec.command`) 
  will be called at Jenkins startup 
  with an integer representing the backup number to restore as first and only argument
  (can be overridden using `spec.restore.recoveryOnce`)

## Example AWS S3 backup using the CLI

This example shows abbreviated version of a simple AWS S3 backup implementation
using: `aws-cli`, `bash` and `kube2iam`. 

In addition to your normal `Jenkins` `CustomResource` some additional settings 
for backup and restore are required, e.g.:

```yaml
kind: Jenkins
apiVersion: jenkins.io/v1alpha1
metadata:
  name: example
  namespace: jenkins
spec:
  master:
    masterAnnotations:
      iam.amazonaws.com/role: "my-example-backup-role" # tell kube2iam where the AWS IAM role is
    containers:
      - name: jenkins-master
        ...
      - name: backup # container responsible for backup and restore
        image: quay.io/virtuslab/aws-cli:1.16.263-2
        workingDir: /home/user/bin/
        command: # our container entry point
          - sleep
          - infinity
        env:
          - name: BACKUP_BUCKET
            value: my-example-bucket # the S3 bucket name to use
          - name: BACKUP_PATH
            value: my-backup-path # the S3 bucket path prefix to use
          - name: JENKINS_HOME
            value: /jenkins-home # the path to mount jenkins home dir in the backup container
        volumeMounts:
          - mountPath: /jenkins-home # Jenkins home volume
            name: jenkins-home
          - mountPath: /home/user/bin/backup.sh
            name: backup-scripts
            subPath: backup.sh
            readOnly: true
          - mountPath: /home/user/bin/restore.sh
            name: backup-scripts
            subPath: restore.sh
            readOnly: true
    volumes:
      - name: backup-scripts
        configMap:
          defaultMode: 0754
          name: jenkins-operator-backup-s3
    securityContext: # make sure both containers use the same UID and GUID
      runAsUser: 1000
      fsGroup: 1000
  ...
  backup:
    containerName: backup # container name responsible for backup
    interval: 3600 # how often make a backup in seconds
    makeBackupBeforePodDeletion: true # trigger backup just before deleting the pod
    action:
      exec:
        command:
          # this command is invoked on "backup" container to create a backup,
          # <backup_number> is passed by operator,
          # for example /home/user/bin/backup.sh <backup_number>
          - /home/user/bin/backup.sh
  restore:
    containerName: backup # container name is responsible for restore backup
    action:
      exec:
        command:
          # this command is invoked on "backup" container to restore a backup,
          # <backup_number> is passed by operator
          # for example /home/user/bin/restore.sh <backup_number>
          - /home/user/bin/restore.sh
#    recoveryOnce: <backup_number> # if want to restore specific backup configure this field and then Jenkins will be restarted and desired backup will be restored
```

The actual backup and restore scripts will be provided in a `ConfigMap`:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: jenkins-operator-backup-s3
  namespace: jenkins
  labels:
    app: jenkins-operator
data:
  backup.sh: |-
    #!/bin/bash -xeu
    [[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1;
    [[ -z "${BACKUP_BUCKET}" ]] && echo "Required 'BACKUP_BUCKET' env not set" && exit 1;
    [[ -z "${BACKUP_PATH}" ]] && echo "Required 'BACKUP_PATH' env not set" && exit 1;
    [[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

    backup_number=$1
    echo "Running backup #${backup_number}"

    BACKUP_TMP_DIR=$(mktemp -d)
    tar -C ${JENKINS_HOME} -czf "${BACKUP_TMP_DIR}/${backup_number}.tar.gz" --exclude jobs/*/workspace* -c jobs && \

    aws s3 cp ${BACKUP_TMP_DIR}/${backup_number}.tar.gz s3://${BACKUP_BUCKET}/${BACKUP_PATH}/${backup_number}.tar.gz
    echo Done

  restore.sh: |-
    #!/bin/bash -xeu
    [[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1
    [[ -z "${BACKUP_BUCKET}" ]] && echo "Required 'BACKUP_BUCKET' env not set" && exit 1;
    [[ -z "${BACKUP_PATH}" ]] && echo "Required 'BACKUP_PATH' env not set" && exit 1;
    [[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

    backup_number=$1
    echo "Running restore #${backup_number}"

    BACKUP_TMP_DIR=$(mktemp -d)
    aws s3 cp s3://${BACKUP_BUCKET}/${BACKUP_PATH}/${backup_number}.tar.gz ${BACKUP_TMP_DIR}/${backup_number}.tar.gz

    tar -C ${JENKINS_HOME} -zxf "${BACKUP_TMP_DIR}/${backup_number}.tar.gz"
    echo Done
```

In our example we will use S3 bucket lifecycle policy to keep
the number of backups under control, e.g. Cloud Formation fragment:
```yaml
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-example-bucket
      ...
      LifecycleConfiguration:
        Rules:
          - Id: BackupCleanup
            Status: Enabled
            Prefix: my-backup-path
            ExpirationInDays: 7
            NoncurrentVersionExpirationInDays: 14
            AbortIncompleteMultipartUpload:
              DaysAfterInitiation: 3
```
