---
title: "Diagnostics"
linkTitle: "Diagnostics"
weight: 40
date: 2019-08-05
description: >
  How to deal with Jenkins Operator problems
---


Turn on debug in **Jenkins Operator** deployment:

```bash
sed -i 's|\(args:\).*|\1\ ["--debug"\]|' deploy/operator.yaml
kubectl apply -f deploy/operator.yaml
```

Watch Kubernetes events:

```bash
kubectl get events --sort-by='{.lastTimestamp}'
```

Verify Jenkins master logs:

```bash
kubectl logs -f jenkins-<cr_name>
```

Verify the `jenkins-operator` logs:

```bash
kubectl logs deployment/jenkins-operator
```

## Troubleshooting

Delete the Jenkins master pod and wait for the new one to come up:

```bash
kubectl delete pod jenkins-<cr_name>
```
