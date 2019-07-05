#!/usr/bin/env bash

export GOPATH=/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
export GO111MODULE=on

kubectl config set-cluster minikube --server=https://$MINIKUBE_IP:8443 \
      --certificate-authority=/minikube/ca.crt && \
    kubectl config set-credentials minikube --certificate-authority=/root/.minikube/ca.crt \
      --client-key=/minikube/client.key \
      --client-certificate=/minikube/client.crt && \
    kubectl config set-context minikube --cluster=minikube --user=minikube && \
    kubectl config use-context minikube

make go-dependencies
ln -s $GOPATH/src/github.com/jenkinsci/kubernetes-operator/vendor/k8s.io $GOPATH/src/k8s.i
ln -s $GOPATH/src/github.com/jenkinsci/kubernetes-operator/vendor/sigs.k8s.io $GOPATH/src/sigs.k8s.io

bash