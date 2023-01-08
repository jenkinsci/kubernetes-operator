---
title: "Configuring backup and restore"
linkTitle: "Configuring backup and restore"
weight: 5
date: 2023-01-08
description: >
  Prevent loss of job history
---

> Because of Jenkins Operator's architecture, the configuration of Jenkins should be done using ConfigurationAsCode 
> or GroovyScripts and jobs should be defined as SeedJobs. It means that there is no point in backing up any job configuration 
> up. Therefore, the backup script makes a copy of jobs history only.

Backup and restore is done by a container sidecar.

### PVC

#### Create PVC

Save to the file named pvc.yaml:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: <pvc_name>
  namespace: <namespace>
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
```

Run the following command:
```bash
$ kubectl -n <namespace> create -f pvc.yaml
```

#### Configure Jenkins CR

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: jenkins-cr
spec:
  jenkinsAPISettings:
    authorizationStrategy: createUser
  master:
    securityContext:
      runAsUser: 1000
      fsGroup: 1000
    disableCSRFProtection: false
    containers:
      - name: jenkins-master
        image: jenkins/jenkins:2.277.4-lts-alpine
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 1500m
            memory: 3Gi
          requests:
            cpu: "1"
            memory: 500Mi
      - name: backup # container responsible for the backup and restore
        env:
          - name: BACKUP_DIR
            value: /backup
          - name: JENKINS_HOME
            value: /jenkins-home
          - name: BACKUP_COUNT
            value: "3" # keep only the 2 most recent backups
        image: virtuslab/jenkins-operator-backup-pvc:v0.1.1 # look at backup/pvc directory
        imagePullPolicy: IfNotPresent
        volumeMounts:
          - mountPath: /jenkins-home # Jenkins home volume
            name: jenkins-home
          - mountPath: /backup # backup volume
            name: backup
        resources:
          limits:
            cpu: 1000m
            memory: 3Gi
          requests:
            cpu: "1"
            memory: 500Mi
    volumes:
      - name: backup # PVC volume where backups will be stored
        persistentVolumeClaim:
          claimName: <pvc_name>
  backup:
    containerName: backup # container name is responsible for backup
    action:
      exec:
        command:
          - /home/user/bin/backup.sh # this command is invoked on "backup" container to make backup, for example /home/user/bin/backup.sh <backup_number>, <backup_number> is passed by operator
    interval: 30 # how often make backup in seconds
    makeBackupBeforePodDeletion: true # make a backup before pod deletion
  restore:
    containerName: backup # container name is responsible for restore backup
    action:
      exec:
        command:
          - /home/user/bin/restore.sh # this command is invoked on "backup" container to make restore backup, for example /home/user/bin/restore.sh <backup_number>, <backup_number> is passed by operator
    #recoveryOnce: <backup_number> # if want to restore specific backup configure this field and then Jenkins will be restarted and desired backup will be restored
    getLatestAction:
      exec:
        command:
          - /home/user/bin/get-latest.sh # this command is invoked on "backup" container to get last backup number before pod deletion; not having it in the CR may cause loss of data
```
