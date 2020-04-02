#!/bin/bash

# Download Helm
wget https://get.helm.sh/helm-v3.1.2-linux-amd64.tar.gz -O /tmp/helm.tar.gz
tar xzf /tmp/helm.tar.gz -C /tmp --strip-components=1
chmod +x /tmp/helm

# Lint helm
/tmp/helm lint ./chart/jenkins-operator