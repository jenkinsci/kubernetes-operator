---
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 3
date: 2019-08-05
description: >
    Jenkins security and hardening out of the box
---

This document helps you to state the reason for an error in the Jenkins Operator, which is the first step in solving it.

## Operator logs
Jenkins Operator can provide some useful logs. To get them, run:
```bash
$ kubectl logs <controller-manager-pod-name> -f 
```

In the logs look for WARNING, ERROR and SEVERE keywords.

## Jenkins logs

If the container is in a CrashLoopBackOff, the fault is in the Jenkins itself.
If the Operator is constantly terminating the pod with ‘missing-plugins’ messages that means the plugins lost compatibility
with the Jenkins image and their version need to be updated.
To learn more about the possible error, check the state of the pod:

```bash
$ kubectl -n <namespace-name> get po <name-of-the-jenkins-pod> -w
```
or
```bash
$ kubectl -n <namespace-name> describe po <name-of-the-jenkins-pod>
```
and check the logs from the Jenkins container:
```bash
$ kubectl -n <namespace-name> logs <jenkins-pod> <jenkins-master> -f 
```


## Kubernetes Events

Sometimes Events provide a great dose of information, especially in the case some Kubernetes resource doesn’t want to become Ready.
To obtain the events in your Cluster run:

```bash
$ kubectl -n <namespace> get events --sort-by='{.lastTimestamp}'
```

## Quick soft reset
You can always kill the Jenkins pod and wait for it to come up again. All the version-controlled configurations will be downloaded again
and the rest will be discarded. Chances are the buggy part will be gone.

```bash
$ kubectl delete pod <jenkins-pod>
```

## Operator debug mode
If you need to access additional logs from the Operator, you can run it in debug mode. To do that, add ``"--debug""``
argument to jenkins-operator container args in your Operator deployment.